package linter

import (
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/parser"
	"github.com/stelligent/config-lint/assertion"
	"io/ioutil"
	"regexp"
)

type (
	// TerraformResourceLoader converts Terraform configuration files into JSON objects
	TerraformResourceLoader struct{}

	// TerraformLoadResult collects all the returns value for parsing an HCL string
	TerraformLoadResult struct {
		Resources []interface{}
		Variables map[string]interface{}
		AST       *ast.File
	}
)

func parsePolicy(templateResource interface{}) (map[string]interface{}, error) {
	firstResource := templateResource.([]interface{})[0] // FIXME does this array always have 1 element?
	properties := firstResource.(map[string]interface{})
	for _, attribute := range []string{"assume_role_policy", "policy"} {
		if policyAttribute, hasPolicyString := properties[attribute]; hasPolicyString {
			if policyString, isString := policyAttribute.(string); isString {
				var policy interface{}
				err := json.Unmarshal([]byte(policyString), &policy)
				if err != nil {
					return properties, err
				}
				properties[attribute] = policy
			}
		}
	}
	return properties, nil
}

func loadHCL(filename string) (TerraformLoadResult, error) {
	result := TerraformLoadResult{
		Resources: []interface{}{},
		Variables: map[string]interface{}{},
	}
	template, err := ioutil.ReadFile(filename)
	if err != nil {
		return result, nil
	}

	result.AST, _ = parser.Parse(template)

	var v interface{}
	err = hcl.Unmarshal([]byte(template), &v)
	if err != nil {
		return result, err
	}
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return result, err
	}
	assertion.Debugf("LoadHCL: %s\n", string(jsonData))

	var hclData interface{}
	err = yaml.Unmarshal(jsonData, &hclData)
	if err != nil {
		return result, err
	}
	m := hclData.(map[string]interface{})
	result.Variables = loadVariables(m["variable"])
	assertion.Debugf("Variables: %v\n", result.Variables)
	if m["resource"] != nil {
		result.Resources = append(result.Resources, m["resource"].([]interface{})...)
	}
	return result, nil
}

func loadVariables(data interface{}) map[string]interface{} {
	variables := map[string]interface{}{}
	if data == nil {
		return variables
	}
	list := data.([]interface{})
	for _, entry := range list {
		m := entry.(map[string]interface{})
		for key, value := range m {
			variables[key] = extractDefault(value)
		}
	}
	return variables
}

func extractDefault(value interface{}) interface{} {
	list := value.([]interface{})
	var defaultValue interface{} = ""
	for _, entry := range list {
		m := entry.(map[string]interface{})
		defaultValue = m["default"]
	}
	return defaultValue
}

func getResourceLineNumber(resourceType, resourceID, filename string, root *ast.File) int {
	resourceItems := root.Node.(*ast.ObjectList).Filter("resource", resourceType, resourceID).Items
	if len(resourceItems) > 0 {
		resourcePos := resourceItems[0].Val.Pos()
		assertion.Debugf("Position %s %s:%d\n", resourceID, filename, resourcePos.Line)
		return resourcePos.Line
	}
	return 0
}

// Load parses an HCL file into a collection or Resource objects
func (l TerraformResourceLoader) Load(filename string) ([]assertion.Resource, error) {
	resources := make([]assertion.Resource, 0)
	result, err := loadHCL(filename)
	if err != nil {
		return resources, err
	}
	for _, resource := range result.Resources {
		for resourceType, templateResources := range resource.(map[string]interface{}) {
			if templateResources != nil {
				for _, templateResource := range templateResources.([]interface{}) {
					for resourceID, templateResource := range templateResource.(map[string]interface{}) {
						properties, err := parsePolicy(templateResource)
						if err != nil {
							return resources, err
						}
						resolvedProperties := replaceVariables(properties, result.Variables)
						lineNumber := getResourceLineNumber(resourceType, resourceID, filename, result.AST)
						tr := assertion.Resource{
							ID:         resourceID,
							Type:       resourceType,
							Properties: resolvedProperties,
							Filename:   filename,
							LineNumber: lineNumber,
						}
						resources = append(resources, tr)
					}
				}
			}
		}
	}
	return resources, nil
}

func replaceVariables(templateResource map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	for key, value := range templateResource {
		if s, ok := value.(string); ok { // FIXME is not a string, need to recurse
			templateResource[key] = resolveValue(s, variables)
		}
	}
	return templateResource
}

func resolveValue(s string, variables map[string]interface{}) string {
	pattern := "[$][{]var[.](?P<name>.*)[}]"
	re, _ := regexp.Compile(pattern)
	match := re.FindStringSubmatch(s)
	if len(match) == 0 {
		return s
	}
	if replacementValue, ok := variables[match[1]].(string); ok {
		assertion.Debugf("Replacing %s with %v\n", s, variables[match[1]])
		return re.ReplaceAllString(s, replacementValue)
	}
	return s
}
