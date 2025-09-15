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

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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
	RequiredCreateDependencies strList
	OptionalCreateDependencies strList
	AllCreateDependencies      strList
	ImportDependencies         strList
}

type strList []string

func (i *strList) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *strList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var interactive bool

	fields := templateFields{}
	flag.StringVar(&fields.Kind, "kind", "", "The kind of the new resource. Required.")
	flag.StringVar(&fields.GophercloudClient, "gophercloud-client", "",
		"The gophercloud client function to use. Required.")
	flag.StringVar(&fields.GophercloudModule, "gophercloud-module", "",
		"The gophercloud module to use. Required.")
	flag.StringVar(&fields.GophercloudType, "gophercloud-type", "",
		"The gophercloud type to use. Defaults to the resource Kind.")
	flag.StringVar(&fields.OpenStackJSONObject, "openstack-json-object", "",
		"The name of the object in OpenStack's json responses.")
	flag.IntVar(&fields.AvailablePollingPeriod, "available-polling-period", 0,
		"The available polling period in seconds. Defaults to 0 (no polling).")
	flag.IntVar(&fields.DeletingPollingPeriod, "deleting-polling-period", 0,
		"The deleting polling period in seconds. Defaults to 0 (no polling).")
	flag.Var(&fields.RequiredCreateDependencies, "required-create-dependency",
		"A required create dependency. Can be specified multiple times.")
	flag.Var(&fields.OptionalCreateDependencies, "optional-create-dependency",
		"An optional create dependency. Can be specified multiple times.")
	flag.Var(&fields.ImportDependencies, "import-dependency",
		"An import filter dependency. Can be specified multiple times.")
	flag.BoolVar(&interactive, "interactive", true, "Whether to run interactively.")
	flag.Parse()

	if fields.Kind == "" {
		fields.Kind = getUserInput("What is the Kind of this resource? For instance: Volume, FloatingIP, ...", interactive)
		if fields.Kind == "" {
			fmt.Fprintln(os.Stderr, "-kind option is required")
			os.Exit(1)
		}
	}

	if fields.GophercloudClient == "" {
		fields.GophercloudClient = getUserInput(
			"What is the gophercloud function used to instantiate a client? For instance: NewBlockStorageV3", interactive)
		if fields.GophercloudClient == "" {
			fmt.Fprintln(os.Stderr, "-gophercloud-client option is required")
			os.Exit(1)
		}
	}

	if fields.GophercloudModule == "" {
		fields.GophercloudModule = getUserInput(
			"What is the gophercloud module? "+
				"For instance: github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes", interactive)
		if fields.GophercloudModule == "" {
			fmt.Fprintln(os.Stderr, "-gophercloud-module option is required")
			os.Exit(1)
		}
	}

	if fields.GophercloudType == "" {
		fields.GophercloudType = getUserInput("What is the gophercloud type? If unset, we'll use "+fields.Kind, interactive)
		if fields.GophercloudType == "" {
			fields.GophercloudType = fields.Kind
		}
	}

	if fields.OpenStackJSONObject == "" {
		jsonObjectName := camelToSnake(fields.Kind)
		fields.OpenStackJSONObject = getUserInput(
			"What is the name of the object in OpenStack json responses? "+
				"If unset, we'll use "+jsonObjectName, interactive)
		if fields.OpenStackJSONObject == "" {
			fields.OpenStackJSONObject = jsonObjectName
		}
	}

	if fields.AvailablePollingPeriod == 0 {
		answer := strings.ToLower(getUserInput("Is the OpenStack resource available right away after creation?", interactive))
		if answer == "y" || answer == "yes" || answer == "" {
			fields.AvailablePollingPeriod = 0
		} else {
			fields.AvailablePollingPeriod = 15
		}
	}

	if fields.DeletingPollingPeriod == 0 {
		answer := strings.ToLower(getUserInput("Is the OpenStack resource deleted right away after deletion? "+
			"If the resource enters a deleting phase, answer no.", interactive))
		if answer == "y" || answer == "yes" || answer == "" {
			fields.DeletingPollingPeriod = 0
		} else {
			fields.DeletingPollingPeriod = 15
		}
	}

	if len(fields.RequiredCreateDependencies) == 0 {
		answer := getUserInput("Does this resource have required dependencies upon creation? "+
			"List all the resources it must depend on. "+
			"Provide this as a comma-separated list, for example: Subnet, Port, Project", interactive)
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.RequiredCreateDependencies = append(fields.RequiredCreateDependencies, trimmedDep)
			}
		}
	}

	if len(fields.OptionalCreateDependencies) == 0 {
		answer := getUserInput("Does this resource have optional dependencies upon creation? "+
			"List all the resources it optionally depend on. "+
			"Provide this as a comma-separated list, for example: Subnet, Port, Project", interactive)
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.OptionalCreateDependencies = append(fields.OptionalCreateDependencies, trimmedDep)
			}
		}
	}

	if len(fields.ImportDependencies) == 0 {
		answer := getUserInput("Does this resource have dependencies upon import? "+
			"List all the resources it can depend on. "+
			"Provide this as a comma-separated list, for example: Subnet, Port, Project", interactive)
		dependencies := strings.Split(answer, ",")
		for _, dep := range dependencies {
			trimmedDep := strings.TrimSpace(dep)
			if trimmedDep != "" {
				fields.ImportDependencies = append(fields.ImportDependencies, trimmedDep)
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
			if file.Name() == "import-dependency" && len(resource.ImportDependencies) == 0 {
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
			"lower":     strings.ToLower,
			"camelCase": toCamelCase,
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

func getUserInput(question string, interactive bool) string {
	if !interactive {
		return ""
	}

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

// toCamelCase converts a string to camelCase.
// From https://stackoverflow.com/questions/70083837/how-to-convert-a-string-to-camelcase-in-go
func toCamelCase(s string) string {
	// Remove all characters that are not alphanumeric or spaces or underscores
	s = regexp.MustCompile("[^a-zA-Z0-9_ ]+").ReplaceAllString(s, "")

	// Replace all underscores with spaces
	s = strings.ReplaceAll(s, "_", " ")

	// Title case s
	s = cases.Title(language.AmericanEnglish, cases.NoLower).String(s)

	// Remove all spaces
	s = strings.ReplaceAll(s, " ", "")

	// Lowercase the first letter
	if len(s) > 0 {
		s = strings.ToLower(s[:1]) + s[1:]
	}

	return s
}
