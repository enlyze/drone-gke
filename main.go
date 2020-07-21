package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
)

type ServiceAccount struct {
	ProjectID string `json:"project_id"`
}

var (
	// Version is set at compile time.
	version string
	// Build revision is set at compile time.
	rev string
)

const (
	gcloudCmd      = "gcloud"
	kubectlCmdName = "kubectl"
	timeoutCmd     = "timeout"
	echoCmd        = "echo"

	nsPath           = "/tmp/namespace.json"
	templateBasePath = "/tmp"
)

// default to kubectlCmdName, can be overriden via kubectl-version param
var kubectlCmd = kubectlCmdName
var extraKubectlVersions = strings.Split(os.Getenv("EXTRA_KUBECTL_VERSIONS"), " ")
var nsTemplate = `
---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`

var invalidNameRegex = regexp.MustCompile(`[^a-z0-9\.\-]+`)

func main() {
	err := wrapMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "dry-run",
			Usage:   "do not apply the Kubernetes manifests to the API server",
			EnvVars: []string{"PLUGIN_DRY_RUN"},
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Usage:   "dump available vars and the generated Kubernetes manifest, keeping secrets hidden",
			EnvVars: []string{"PLUGIN_VERBOSE"},
		},
		&cli.StringFlag{
			Name:    "project",
			Usage:   "GCP project name (default: interpreted from JSON credentials)",
			EnvVars: []string{"PLUGIN_PROJECT"},
		},
		&cli.StringFlag{
			Name:    "zone",
			Usage:   "zone of the container cluster",
			EnvVars: []string{"PLUGIN_ZONE"},
		},
		&cli.StringFlag{
			Name:    "region",
			Usage:   "region of the container cluster",
			EnvVars: []string{"PLUGIN_REGION"},
		},
		&cli.StringFlag{
			Name:    "cluster-name",
			Usage:   "name of the container cluster",
			EnvVars: []string{"PLUGIN_CLUSTER_NAME"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Usage:   "Kubernetes namespace to operate in",
			EnvVars: []string{"PLUGIN_NAMESPACE"},
		},
		&cli.StringFlag{
			Name:    "kube-template",
			Usage:   "template for Kubernetes resources, e.g. Deployments",
			EnvVars: []string{"PLUGIN_TEMPLATE"},
			Value:   ".kube.yml",
		},
		&cli.StringFlag{
			Name:    "vars",
			Usage:   "variables to use while templating manifests in `JSON` format",
			EnvVars: []string{"PLUGIN_VARS"},
		},
		&cli.BoolFlag{
			Name:    "expand-env-vars",
			Usage:   "expand environment variables contents on vars",
			EnvVars: []string{"PLUGIN_EXPAND_ENV_VARS"},
		},
		&cli.StringFlag{
			Name:    "drone-build-number",
			Usage:   "Drone build number",
			EnvVars: []string{"DRONE_BUILD_NUMBER"},
		},
		&cli.StringFlag{
			Name:    "drone-commit",
			Usage:   "Git commit hash",
			EnvVars: []string{"DRONE_COMMIT"},
		},
		&cli.StringFlag{
			Name:    "drone-branch",
			Usage:   "Git branch",
			EnvVars: []string{"DRONE_BRANCH"},
		},
		&cli.StringFlag{
			Name:    "drone-tag",
			Usage:   "Git tag",
			EnvVars: []string{"DRONE_TAG"},
		},
		&cli.StringSliceFlag{
			Name:    "wait-deployments",
			Usage:   "list of Deployments to wait for successful rollout using kubectl rollout status in `JSON` format",
			EnvVars: []string{"PLUGIN_WAIT_DEPLOYMENTS"},
		},
		&cli.IntFlag{
			Name:    "wait-seconds",
			Usage:   "if wait-deployments is set, number of seconds to wait before failing the build",
			EnvVars: []string{"PLUGIN_WAIT_SECONDS"},
			Value:   0,
		},
		&cli.StringFlag{
			Name:    "kubectl-version",
			Usage:   "optional - version of kubectl binary to use, e.g. 1.14",
			EnvVars: []string{"PLUGIN_KUBECTL_VERSION"},
		},
	}
}

func wrapMain() error {
	if version == "" {
		version = "x.x.x"
	}

	if rev == "" {
		rev = "[unknown]"
	}

	fmt.Printf("Drone GKE Plugin built from %s\n", rev)

	app := cli.NewApp()
	app.Name = "gke plugin"
	app.Usage = "gke plugin"
	app.Action = run
	app.Version = fmt.Sprintf("%s-%s", version, rev)
	app.Flags = getAppFlags()
	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}

func run(c *cli.Context) error {
	// Check required params
	if err := checkParams(c); err != nil {
		return err
	}

	// Use project if explicitly stated, otherwise infer from the service account token.
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	project := c.String("project")
	if project == "" {
		log("Parsing Project ID from credentials\n")
		infered, err := projectFromServiceAccount(credentialsPath)
		if err != nil {
			return fmt.Errorf("Could not infer project from credentials: %s", err.Error())
		}

		project = infered
	}

	// Use custom kubectl version if provided.
	kubectlVersion := c.String("kubectl-version")
	if kubectlVersion != "" {
		kubectlCmd = fmt.Sprintf("%s.%s", kubectlCmdName, kubectlVersion)
	}

	// Parse variables and secrets
	vars, err := parseVars(c)
	if err != nil {
		return err
	}

	// Setup execution environment
	environ := os.Environ()
	runner := NewBasicRunner("", environ, os.Stdout, os.Stderr)

	// Auth with gcloud and fetch kubectl credentials
	if err := fetchCredentials(c, credentialsPath, project, runner); err != nil {
		return err
	}

	templateString, err := readTemplate()
	if err != nil || templateString == "" {
		log("Error: error reading template %s", err.Error())
		return err
	}

	// Build template data maps
	templateData, err := templateData(c, project, vars)
	if err != nil {
		return err
	}

	// Print variables and secret keys
	if c.Bool("verbose") {
		dumpData(os.Stdout, "VARIABLES AVAILABLE FOR ALL TEMPLATES", templateData)
	}

	// Render manifest templates
	manifest, err := renderTemplates(c, templateString, templateData)

	if err != nil {
		return err
	}

	// kubectl version
	if err := printKubectlVersion(runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Set namespace and ensure it exists
	if err := setNamespace(c, project, runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Apply manifests
	if err := applyManifest(c, manifest, runner); err != nil {
		return fmt.Errorf("Error (kubectl output redacted): %s\n", err)
	}

	// Wait for rollout to finish
	if err := waitForRollout(c, runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// checkParams checks required params
func checkParams(c *cli.Context) error {

	if c.String("zone") == "" && c.String("region") == "" {
		return fmt.Errorf("Missing required param: at least one of region or zone must be specified")
	}

	if c.String("zone") != "" && c.String("region") != "" {
		return fmt.Errorf("Invalid params: at most one of region or zone may be specified")
	}

	if c.String("cluster-name") == "" {
		return fmt.Errorf("Missing required param: cluster-name")
	}

	if err := validateKubectlVersion(c, extraKubectlVersions); err != nil {
		return err
	}

	return nil
}

// validateKubectlVersion tests whether a given version is valid within the current environment
func validateKubectlVersion(c *cli.Context, availableVersions []string) error {
	kubectlVersionParam := c.String("kubectl-version")
	// using the default version
	if kubectlVersionParam == "" {
		return nil
	}

	// using a custom version but no extra versions are available
	if len(availableVersions) == 0 {
		return fmt.Errorf("Invalid param: kubectl-version was set to %s but no extra kubectl versions are available", kubectlVersionParam)
	}

	// using a custom version ...
	// return nil if included in available extra versions; error otherwise
	for _, availableVersion := range availableVersions {
		if kubectlVersionParam == availableVersion {
			return nil
		}
	}
	return fmt.Errorf("Invalid param kubectl-version: %s must be one of %s", kubectlVersionParam, strings.Join(availableVersions, ", "))
}

// projectFromServiceAccount gets project id from service account
func projectFromServiceAccount(credentialsPath string) (string, error) {
	credentials, err := os.Open(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("Could not open file: %v", err.Error())
	}
	defer credentials.Close()

	credentialsBytes, _ := ioutil.ReadAll(credentials)

	t := ServiceAccount{}
	err = json.Unmarshal(credentialsBytes, &t)
	if err != nil {
		return "", fmt.Errorf("Could not unmarshal credentials file: %v", err.Error())
	}

	return t.ProjectID, nil
}

// parseVars parses vars (in JSON) and returns a map
func parseVars(c *cli.Context) (map[string]interface{}, error) {
	// Parse variables.
	vars := make(map[string]interface{})
	varsJSON := c.String("vars")
	if varsJSON != "" {
		if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
			return nil, fmt.Errorf("Error parsing vars: %s\n", err)
		}
	}

	return vars, nil
}

// fetchCredentials authenticates with gcloud and fetches credentials for kubectl
func fetchCredentials(c *cli.Context, credentialsPath string, project string, runner Runner) error {
	err := runner.Run(gcloudCmd, "auth", "activate-service-account", "--key-file", credentialsPath)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	getCredentialsArgs := []string{
		"container",
		"clusters",
		"get-credentials", c.String("cluster-name"),
		"--project", project,
	}

	// build --zone / --region arguments based on parameters provided to plugin
	// checkParams requires at least one of zone or region to be provided and prevents use of both at the same time
	if c.String("zone") != "" {
		getCredentialsArgs = append(getCredentialsArgs, "--zone", c.String("zone"))
	}

	if c.String("region") != "" {
		getCredentialsArgs = append(getCredentialsArgs, "--region", c.String("region"))
	}

	err = runner.Run(gcloudCmd, getCredentialsArgs...)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

func readTemplate() (string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		fmt.Println("The command is intended to work with pipes.")
		fmt.Println("Usage: cat kube.yaml | drone-gke")
		return "", errors.New("found no stdin")
	}

	reader := bufio.NewReader(os.Stdin)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// templateData builds template and data maps
func templateData(c *cli.Context, project string, vars map[string]interface{}) (map[string]interface{}, error) {
	// Built-in template vars
	templateData := map[string]interface{}{
		"BUILD_NUMBER": c.String("drone-build-number"),
		"COMMIT":       c.String("drone-commit"),
		"BRANCH":       c.String("drone-branch"),
		"TAG":          c.String("drone-tag"),

		// Misc useful stuff.
		// Note that secrets (including the GCP token) are excluded
		"project":      project,
		"zone":         c.String("zone"),
		"cluster-name": c.String("cluster-name"),
		"namespace":    c.String("namespace"),
	}

	// Add variables to data used for rendering both templates.
	for k, v := range vars {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := templateData[k]; ok {
			return nil, fmt.Errorf("Error: var %q shadows existing var\n", k)
		}

		if c.Bool("expand-env-vars") {
			if rawValue, ok := v.(string); ok {
				v = os.ExpandEnv(rawValue)
			}
		}

		templateData[k] = v
	}

	return templateData, nil
}

// renderTemplates renders templates, writes into files and returns rendered template paths
func renderTemplates(c *cli.Context, templateString string, templateData map[string]interface{}) (string, error) {

	// Parse the template.
	tmpl, err := template.New("manifast.yaml").Option("missingkey=error").Parse(templateString)
	if err != nil {
		return "", fmt.Errorf("Error parsing template: %s\n", err)
	}

	// Generate the manifest.
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, templateData)
	if err != nil {
		return "", fmt.Errorf("Error rendering deployment manifest from template: %s\n", err)
	}

	return rendered.String(), nil
}

// printKubectlVersion runs kubectl version
func printKubectlVersion(runner Runner) error {
	return runner.Run(kubectlCmd, "version")
}

// setNamespace sets namespace of current kubectl context and ensure it exists
func setNamespace(c *cli.Context, project string, runner Runner) error {
	namespace := c.String("namespace")
	if namespace == "" {
		return nil
	}

	//replace invalid char in namespace
	namespace = strings.ToLower(namespace)
	namespace = invalidNameRegex.ReplaceAllString(namespace, "-")

	// Set the execution namespace.
	log("Configuring kubectl to the %s namespace\n", namespace)

	// set cluster location segment based on parameters provided to plugin
	// checkParams requires at least one of zone or region to be provided and prevents use of both at the same time
	clusterLocation := ""
	if c.String("zone") != "" {
		clusterLocation = c.String("zone")
	}

	if c.String("region") != "" {
		clusterLocation = c.String("region")
	}

	context := strings.Join([]string{"gke", project, clusterLocation, c.String("cluster-name")}, "_")

	if err := runner.Run(kubectlCmd, "config", "set-context", context, "--namespace", namespace); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Write the namespace manifest to a tmp file for application.
	nsManifest := fmt.Sprintf(nsTemplate, namespace)
	// Ensure the namespace exists, without errors (unlike `kubectl create namespace`).
	log("Ensuring the %s namespace exists\n", namespace)

	cmd, args := applyCmd(c.Bool("dry-run"))
	if err := runner.RunWithPipedInput(nsManifest, cmd, args...); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// applyManifests applies manifests using kubectl apply
func applyManifest(c *cli.Context, manifest string, runner Runner) error {

	// If it is not a dry run, do a dry run first to validate Kubernetes manifests.
	log("Validating Kubernetes manifests with a dry-run\n")

	if !c.Bool("dry-run") {
		cmd, args := applyCmd(true)
		if err := runner.RunWithPipedInput(manifest, cmd, args...); err != nil {
			log("%s", runner.Stdout())
			log("%s", runner.Stderr())
			return fmt.Errorf("Error: %s\n", err)
		}
		log("Applying Kubernetes manifest to the cluster\n")
	}

	// Actually apply Kubernetes manifests.
	cmd, args := applyCmd(c.Bool("dry-run"))
	if err := runner.RunWithPipedInput(manifest, cmd, args...); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// waitForRollout executes kubectl to wait for rollout to complete before continuing
func waitForRollout(c *cli.Context, runner Runner) error {

	namespace := c.String("namespace")
	waitSeconds := c.Int("wait-seconds")
	specs := c.StringSlice("wait-deployments")
	waitDeployments := []string{}
	for _, spec := range specs {
		// default type to "deployment" if not present
		deployment := spec
		if !strings.Contains(spec, "/") {
			deployment = "deployment/" + deployment
		}
		waitDeployments = append(waitDeployments, deployment)
	}

	waitDeploymentsCount := len(waitDeployments)
	counterProgress := ""

	for counter, deployment := range waitDeployments {
		if waitDeploymentsCount > 1 {
			counterProgress = fmt.Sprintf(" %d/%d", counter+1, waitDeploymentsCount)
		}

		log(fmt.Sprintf("Waiting until rollout completes for %s%s\n", deployment, counterProgress))

		command := []string{"rollout", "status", deployment}

		if namespace != "" {
			command = append(command, "--namespace", namespace)
		}

		path := kubectlCmd

		if waitSeconds != 0 {
			command = append([]string{strconv.Itoa(waitSeconds), path}, command...)
			path = timeoutCmd
		}

		if err := runner.Run(path, command...); err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}
	}

	return nil
}

func applyCmd(dryrun bool) (string, []string) {

	args := []string{
		"apply",
		"--record",
	}

	if dryrun {
		args = append(args, "--dry-run")
	}

	args = append(args, "-f", "-")

	return kubectlCmd, args

}

// printTrimmedError prints the last line of stderrbuf to dest
func printTrimmedError(stderrbuf io.Reader, dest io.Writer) {
	var lastLine string
	scanner := bufio.NewScanner(stderrbuf)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	fmt.Fprintf(dest, "%s\n", lastLine)
}

func log(format string, a ...interface{}) {
	fmt.Printf("\n"+format, a...)
}
