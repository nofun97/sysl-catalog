package templategeneration

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/anz-bank/sysl-catalog/pkg/catalogdiagrams"

	"github.com/anz-bank/sysl/pkg/sequencediagram"

	"github.com/anz-bank/sysl/pkg/datamodeldiagram"

	"github.com/anz-bank/sysl/pkg/sysl"
	"github.com/anz-bank/sysl/pkg/syslutil"
)

const (
	md           = ".md"
	ext          = ".svg"
	pageFilename = "README.md"
)

// Package is the second level where apps and endpoints are specified.
type Package struct {
	Parent              *Project
	OutputDir           string
	PackageName         string
	OutputFile          string
	IntegrationDiagrams []*Diagram
	SequenceDiagrams    []*SequenceDiagram // map[appName + endpointName]
	DataModelDiagrams   []*Diagram
}

func (p Package) RegisterIntegrationDiagrams(m *sysl.Module) {

}

func (p Package) RegisterDataModelDiagrams(m *sysl.Module) {

}

// AlphabeticalRows returns an alphabetically sorted list of packages of any project.
func (p Project) AlphabeticalRows() []*Package {
	packages := make([]*Package, 0, len(p.Packages))
	for _, key := range AlphabeticalPackage(p.Packages) {
		packages = append(packages, p.Packages[key])
	}
	return packages
}

// RegisterSequenceDiagrams creates sequence Diagrams from the sysl Module in Project.
func (p Project) RegisterSequenceDiagrams() error {
	for _, key := range AlphabeticalApps(p.Module.Apps) {
		app := p.Module.Apps[key]
		packageName, appName := GetAppPackageName(app)
		if syslutil.HasPattern(app.Attrs, "ignore") {
			p.Log.Infof("Skipping application %s", app.Name)
			continue
		}
		for _, key2 := range AlphabeticalEndpoints(app.Endpoints) {
			endpoint := app.Endpoints[key2]
			if syslutil.HasPattern(endpoint.Attrs, "ignore") {
				p.Log.Infof("Skipping application %s", app.Name)
				continue
			}
			packageD := p.Packages[packageName]
			diagram, err := packageD.SequenceDiagramFromEndpoint(appName, endpoint)
			if err != nil {
				return err
			}
			p.Packages[packageName].SequenceDiagrams = append(packageD.SequenceDiagrams, diagram)
			if p.Packages[packageName].DataModelDiagrams == nil {
				p.Packages[packageName].DataModelDiagrams = []*Diagram{}
			}
		}
	}
	return nil
}

func (p Project) GenerateEndpointDataModel(parentAppName string, t *sysl.Type) string {
	pl := &datamodelCmd{}
	pl.Project = ""
	p.Fs.MkdirAll(pl.Output, os.ModePerm)
	pl.Direct = true
	pl.ClassFormat = "%(classname)"
	spclass := sequencediagram.ConstructFormatParser("", pl.ClassFormat)
	var stringBuilder strings.Builder
	dataParam := &catalogdiagrams.DataModelParam{}
	dataParam.Mod = p.Module

	v := datamodeldiagram.MakeDataModelView(spclass, dataParam.Mod, &stringBuilder, dataParam.Title, "")
	vNew := &catalogdiagrams.DataModelView{
		DataModelView: *v,
	}
	return vNew.GenerateDataView(dataParam, parentAppName, t, p.Module)
}

// SequenceDiagramFromEndpoint generates a sequence diagram from a sysl endpoint
func (p Package) SequenceDiagramFromEndpoint(appName string, endpoint *sysl.Endpoint) (*SequenceDiagram, error) {
	call := fmt.Sprintf("%s <- %s", appName, endpoint.Name)
	var re = regexp.MustCompile(`(?m)\w+\.\w+`)
	var typeName string
	seq, err := CreateSequenceDiagram(p.Parent.Module, call)
	if err != nil {
		return nil, err
	}
	diagram := &SequenceDiagram{}
	diagram.Parent = &p
	diagram.AppName = appName
	diagram.EndpointName = endpoint.Name
	diagram.OutputFileName__ = appName + endpoint.Name
	diagram.OutputDir = path.Join(p.Parent.Output, p.PackageName)
	diagram.DiagramString = seq
	diagram.Diagramtype = diagram_sequence
	diagram.OutputMarkdownFileName = pageFilename
	diagram.OutputDataModel = []*Diagram{}
	diagram.InputDataModel = []*Diagram{}
	for _, param := range endpoint.Param {
		newDiagram := &Diagram{
			Parent:           &p,
			OutputDir:        path.Join(p.Parent.Output, p.PackageName),
			AppName:          appName,
			DiagramString:    p.Parent.GenerateEndpointDataModel(appName, param.Type),
			OutputFileName__: appName + endpoint.Name + "data-model",
			EndpointName:     endpoint.Name,
		}
		diagram.InputDataModel = append(diagram.InputDataModel, newDiagram)
	}
	for _, stmnt := range endpoint.Stmt {
		if ret := stmnt.GetRet(); ret != nil {
			fmt.Println(ret.Payload)
			t := re.FindString(ret.Payload)
			if split := strings.Split(t, "."); len(split) > 1 {
				appName = split[0]
				typeName = split[1]
			}
			typeref := &sysl.Type{
				Type: &sysl.Type_TypeRef{
					TypeRef: &sysl.ScopedRef{
						Ref: &sysl.Scope{Appname: &sysl.AppName{
							Part: []string{appName},
						},
							Path: []string{appName, typeName},
						},
					},
				},
			}
			newDiagram := &Diagram{
				Parent:           &p,
				OutputDir:        path.Join(p.Parent.Output, p.PackageName),
				AppName:          appName,
				DiagramString:    p.Parent.GenerateEndpointDataModel(appName, typeref),
				OutputFileName__: appName + endpoint.Name + "data-model",
				EndpointName:     endpoint.Name,
			}
			diagram.OutputDataModel = append(diagram.OutputDataModel, newDiagram)
		}
	}
	return diagram, nil
}