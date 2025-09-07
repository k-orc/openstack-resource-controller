package main

import (
	"bufio"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// TODO:
// - generate bundle manifest with make (it requires installing openshift-sdk, wich would need to be installed on every CI run. Perhaps not worth it.)
// - Documentation

// Then, one needs to:
// Add to `cmd/resource-generator/main.go` and run `make generate`
// Add the new client to `internal/scope/scope.go`
// Add to `cmd/manager/main.go`
// Implement TODOs
// Run the tests
// Update README

//go:embed data
var data embed.FS

type templateFields struct {
	Kind                       string
	PackageName                string
	GophercloudClient          string
	GophercloudModule          string
	GophercloudPackage         string
	GophercloudType            string
	OpenStackJSONObject        string
	AvailablePollingPeriod     int
	DeletingPollingPeriod      int
	Year                       int
	RequiredCreateDependencies []string
	OptionalCreateDependencies []string
	AllCreateDependencies      []string
	FilterDependencies         []string
}

func main() {
	fields := templateFields{}
	// TODO: permit a full non-interactive experience by supporting all options as flags
	flag.StringVar(&fields.Kind, "kind", "", "The kind of the new resource")
	flag.StringVar(&fields.GophercloudClient, "gophercloud-client", "", "The gophercloud client to use")
	flag.StringVar(&fields.GophercloudModule, "gophercloud-module", "", "The gophercloud module to use")
	flag.StringVar(&fields.GophercloudType, "gophercloud-type", "", "The gophercloud type to use")
	flag.StringVar(&fields.OpenStackJSONObject, "openstack-json-object", "",
		"The name of the object in OpenStack's json responses")
	flag.IntVar(&fields.AvailablePollingPeriod, "available-polling-period", 0, "The available polling period in seconds.")
	flag.IntVar(&fields.DeletingPollingPeriod, "deleting-polling-period", 0, "The deleting polling period in seconds.")
	flag.Parse()

	if fields.Kind == "" {
		fields.Kind = getUserInput("What is the Kind of this resource? For instance: Volume, FloatingIP, ...")
	}

	if fields.GophercloudClient == "" {
		fields.GophercloudClient = getUserInput(
			"What is the gophercloud function used to instantiate a client? For instance: NewBlockStorageV3")
	}

	if fields.GophercloudModule == "" {
		fields.GophercloudModule = getUserInput(
			"What is the gophercloud module? " +
				"For instance: github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes")
	}

	if fields.GophercloudType == "" {
		fields.GophercloudType = getUserInput("What is the gophercloud type? If unset, we'll use " + fields.Kind)
		if fields.GophercloudType == "" {
			fields.GophercloudType = fields.Kind
		}
	}

	if fields.OpenStackJSONObject == "" {
		jsonObjectName := camelToSnake(fields.Kind)
		fields.OpenStackJSONObject = getUserInput(
			"What is the name of the object in OpenStack json responses? " +
				"If unset, we'll use " + jsonObjectName)
		if fields.OpenStackJSONObject == "" {
			fields.OpenStackJSONObject = jsonObjectName
		}
	}

	if fields.AvailablePollingPeriod == 0 {
		answer := strings.ToLower(getUserInput("Is the OpenStack resource available right away after creation?"))
		if answer == "y" || answer == "yes" {
			fields.AvailablePollingPeriod = 0
		} else {
			fields.AvailablePollingPeriod = 15
		}
	}

	if fields.DeletingPollingPeriod == 0 {
		answer := strings.ToLower(getUserInput("Is the OpenStack resource deleted right away after deletion? " +
			"If the resource enters a deleting phase, answer no."))
		if answer == "y" || answer == "yes" {
			fields.DeletingPollingPeriod = 0
		} else {
			fields.DeletingPollingPeriod = 15
		}
	}

	if len(fields.RequiredCreateDependencies) == 0 {
		answer := getUserInput("Does this resource have required dependencies upon creation? " +
			"List all the resources it must depend on. " +
			"Provide this as a comma-separated list, for example: Subnet, Port, Project")
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.RequiredCreateDependencies = append(fields.RequiredCreateDependencies, trimmedDep)
			}
		}
	}

	if len(fields.OptionalCreateDependencies) == 0 {
		answer := getUserInput("Does this resource have optional dependencies upon creation? " +
			"List all the resources it optionally depend on. " +
			"Provide this as a comma-separated list, for example: Subnet, Port, Project")
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.OptionalCreateDependencies = append(fields.OptionalCreateDependencies, trimmedDep)
			}
		}
	}

	if len(fields.FilterDependencies) == 0 {
		answer := getUserInput("Does this resource have dependencies upon import? " +
			"List all the resources it can depend on. " +
			"Provide this as a comma-separated list, for example: Subnet, Port, Project")
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.FilterDependencies = append(fields.FilterDependencies, trimmedDep)
			}
		}
	}

	fields.PackageName = strings.ToLower(fields.Kind)
	fields.GophercloudPackage = path.Base(fields.GophercloudModule)
	fields.Year = time.Now().Year()
	fields.AllCreateDependencies = slices.Concat(fields.RequiredCreateDependencies, fields.OptionalCreateDependencies)

	render("data/api", filepath.Join("api", "v1alpha1"), &fields)
	render("data/client", filepath.Join("internal", "osclients"), &fields)
	render("data/controller", filepath.Join("internal", "controllers", fields.PackageName), &fields)
	render("data/tests", filepath.Join("internal", "controllers", fields.PackageName, "tests"), &fields)
	render("data/samples", filepath.Join("config", "samples"), &fields)
}

func render(srcDir, distDir string, resource *templateFields) {
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		err = os.MkdirAll(distDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	files, err := data.ReadDir(srcDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			if file.Name() == "dependency" && len(resource.OptionalCreateDependencies) == 0 {
				continue
			}
			if file.Name() == "import-dependency" && len(resource.FilterDependencies) == 0 {
				continue
			}
			render(filepath.Join(srcDir, file.Name()),
				filepath.Join(distDir, resource.PackageName+"-"+file.Name()),
				resource)
		}

		if !strings.HasSuffix(file.Name(), ".template") {
			continue
		}

		templatePath := filepath.Join(srcDir, file.Name())
		templateContent, err := data.ReadFile(templatePath)
		if err != nil {
			panic(err)
		}

		tplName := strings.TrimSuffix(file.Name(), ".template")
		switch tplName {
		case "types.go":
			tplName = resource.PackageName + "_" + tplName
		case "client.go":
			tplName = resource.PackageName + ".go"
		case "sample.yaml":
			tplName = "openstack_v1alpha1_" + resource.PackageName + ".yaml"
		}

		var funcMap = template.FuncMap{
			"lower": strings.ToLower,
		}
		tpl := template.Must(template.New(tplName).Funcs(funcMap).Parse(string(templateContent)))

		outfilePath := filepath.Join(distDir, tplName)
		if err := writeTemplate(outfilePath, tpl, resource); err != nil {
			panic(err)
		}
	}
}

func writeTemplate(path string, template *template.Template, resource *templateFields) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return template.Execute(file, resource)
}

func getUserInput(question string) string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("> " + question)
	scanner.Scan()

	response := strings.TrimSpace(scanner.Text())

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		return ""
	}

	return response
}

// camelToSnake converts a camelCase string to snake_case.
func camelToSnake(s string) string {
	// Add an underscore before each uppercase letter that is not at the start of the string.
	// Example: "camelCase" -> "camel_Case"
	// Example: "HTTPRequest" -> "HTTP_Request"
	re1 := regexp.MustCompile("([A-Z])([A-Z][a-z])")
	s = re1.ReplaceAllString(s, "${1}_${2}")

	// Add an underscore before each uppercase letter that is followed by a lowercase letter
	// and is not at the start of the string.
	// Example: "camel_Case" -> "camel_case" (after lowercasing)
	// Example: "HTTP_Request" -> "http_request" (after lowercasing)
	re2 := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = re2.ReplaceAllString(s, "${1}_${2}")

	return strings.ToLower(s)
}
