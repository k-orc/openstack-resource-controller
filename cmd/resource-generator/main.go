package main

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
)

const (
	defaultYear       = "2025"
	defaultAPIVersion = "v1alpha1"
)

//go:embed data/api.template
var api_template string

//go:embed data/adapter.template
var adapter_template string

//go:embed data/controller.template
var controller_template string

//go:embed data/PROJECT.template
var project_template string

//go:embed data/kuttl-test.yaml.template
var kuttl_test_template string

//go:embed data/config-crd-kustomization.yaml.template
var crd_kustomization_template string

//go:embed data/config-samples-kustomization.yaml.template
var samples_kustomization_template string

//go:embed data/internal-osclients-mock-doc.go.template
var mock_doc_template string

type specExtraValidation struct {
	Rule    string
	Message string
}

type additionalPrintColumn struct {
	Name        string
	Type        string
	JSONPath    string
	Description string
}

type templateFields struct {
	APIVersion             string
	Year                   string
	Name                   string
	NameLower              string
	IsNotNamed             bool
	SpecExtraType          string
	StatusExtraType        string
	SpecExtraValidations   []specExtraValidation
	AdditionalPrintColumns []additionalPrintColumn
	// NOTE: this is temporary until we migrate the controllers to use a dedicated client
	// New controllers should not set this field
	ExistingOSClient bool
	// UsesNameAsID indicates that this resource uses its name as the ID instead of a UUID.
	// When true, the UUID validation will be omitted from the Import.ID field.
	// Default is false (uses UUID).
	UsesNameAsID bool
}

var resources []templateFields = []templateFields{
	{
		Name: "Domain",
	},
	{
		Name:             "Flavor",
		ExistingOSClient: true,
	},
	{
		Name: "FloatingIP",
		AdditionalPrintColumns: []additionalPrintColumn{
			{
				Name:        "Address",
				Type:        "string",
				JSONPath:    ".status.resource.floatingIP",
				Description: "Allocated IP address",
			},
		},
		IsNotNamed:       true, // FloatingIP is not named in OpenStack
		ExistingOSClient: true,
	},
	{
		Name: "Image",
		SpecExtraValidations: []specExtraValidation{
			{
				Rule:    "!has(self.__import__) ? has(self.resource.content) : true",
				Message: "resource content must be specified when not importing",
			},
		},
		StatusExtraType:  "ImageStatusExtra",
		ExistingOSClient: true,
	},
	{
		Name:             "Network",
		ExistingOSClient: true,
	},
	{
		Name: "Port",
		AdditionalPrintColumns: []additionalPrintColumn{
			{
				Name:        "Addresses",
				Type:        "string",
				JSONPath:    ".status.resource.fixedIPs[*].ip",
				Description: "Allocated IP addresses",
			},
		},
		ExistingOSClient: true,
	},
	{
		Name:             "Project",
		ExistingOSClient: true,
	},
	{
		Name:             "Router",
		ExistingOSClient: true,
	},
	{
		Name:             "SecurityGroup",
		ExistingOSClient: true,
	},
	{
		Name:             "Server",
		ExistingOSClient: true,
	},
	{
		Name:             "ServerGroup",
		ExistingOSClient: true,
	},
	{
		Name:             "Subnet",
		ExistingOSClient: true,
	},
	{
		Name: "Volume",
	},
	{
		Name: "VolumeType",
	},
	{
		Name: "Service",
	},
}

// These resources won't be generated
var specialResources []templateFields = []templateFields{
	{
		Name:             "RouterInterface",
		ExistingOSClient: true,
	},
}

func main() {
	apiTemplate := template.Must(template.New("api").Parse(api_template))
	adapterTemplate := template.Must(template.New("adapter").Parse(adapter_template))
	controllerTemplate := template.Must(template.New("controller").Parse(controller_template))
	projectTemplate := template.Must(template.New("project").Parse(project_template))
	kuttlTestTemplate := template.Must(template.New("kuttl-test").Parse(kuttl_test_template))
	crdKustomizationTemplate := template.Must(template.New("crd-kustomization").Parse(crd_kustomization_template))
	samplesKustomizationTemplate := template.Must(
		template.New("samples-kustomization").Parse(samples_kustomization_template))
	mockDocTemplate := template.Must(template.New("mock-doc").Parse(mock_doc_template))

	addDefaults(resources)
	addDefaults(specialResources)

	for i := range resources {
		resource := &resources[i]

		apiPath := filepath.Join("api", resource.APIVersion, "zz_generated."+resource.NameLower+"-resource.go")
		if err := writeTemplate(apiPath, apiTemplate, resource); err != nil {
			panic(err)
		}

		controllerDirPath := filepath.Join("internal", "controllers", resource.NameLower)
		if _, err := os.Stat(controllerDirPath); os.IsNotExist(err) {
			err = os.Mkdir(controllerDirPath, 0755)
			if err != nil {
				panic(err)
			}
		}

		adapterPath := filepath.Join(controllerDirPath, "zz_generated.adapter.go")
		if err := writeTemplate(adapterPath, adapterTemplate, resource); err != nil {
			panic(err)
		}

		controllerPath := filepath.Join(controllerDirPath, "zz_generated.controller.go")
		if err := writeTemplate(controllerPath, controllerTemplate, resource); err != nil {
			panic(err)
		}
	}

	// NOTE: some resources needs special handling.
	// Let's add them now and sort the resulting slice alphabetically by resource name.
	allResources := slices.Concat(resources, specialResources)
	sort.Slice(allResources, func(i, j int) bool {
		return allResources[i].Name < allResources[j].Name
	})

	if err := writeTemplate("PROJECT", projectTemplate, allResources); err != nil {
		panic(err)
	}

	if err := writeTemplate("kuttl-test.yaml", kuttlTestTemplate, allResources); err != nil {
		panic(err)
	}

	crdKustomizationPath := filepath.Join("config", "crd", "kustomization.yaml")
	if err := writeTemplate(crdKustomizationPath, crdKustomizationTemplate, allResources); err != nil {
		panic(err)
	}

	samplesKustomizationPath := filepath.Join("config", "samples", "kustomization.yaml")
	if err := writeTemplate(samplesKustomizationPath, samplesKustomizationTemplate, allResources); err != nil {
		panic(err)
	}

	mockDocPath := filepath.Join("internal", "osclients", "mock", "doc.go")
	if err := writeTemplate(mockDocPath, mockDocTemplate, allResources); err != nil {
		panic(err)
	}
}

func addDefaults(resources []templateFields) {
	for i := range resources {
		resource := &resources[i]

		if resource.Year == "" {
			resource.Year = defaultYear
		}

		if resource.APIVersion == "" {
			resource.APIVersion = defaultAPIVersion
		}

		resource.NameLower = strings.ToLower(resource.Name)
	}
}

type ResourceType interface {
	*templateFields | []templateFields
}

func writeTemplate[T ResourceType](path string, tmpl *template.Template, resource T) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	err = writeAutogeneratedHeader(file)
	if err != nil {
		return err
	}

	return tmpl.Execute(file, resource)
}

func writeAutogeneratedHeader(f *os.File) error {
	var commentPrefix string

	switch filepath.Ext(f.Name()) {
	case ".go":
		commentPrefix = "//"
	case ".yaml", ".yml":
		commentPrefix = "#"
	default:
		commentPrefix = "#"
	}

	header := commentPrefix + " Code generated by resource-generator. DO NOT EDIT.\n"
	_, err := f.WriteString(header)

	return err
}
