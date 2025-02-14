/*
Copyright 2022 The KubeOne Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"io"
	"testing"
	"text/template"

	"sigs.k8s.io/yaml"
)

type scenarioUpgrade struct {
	name                 string
	manifestTemplatePath string
	versions             []string
	infra                Infra
}

func (scenario scenarioUpgrade) Title() string { return titleize(scenario.name) }

func (scenario *scenarioUpgrade) SetInfra(infra Infra) {
	scenario.infra = infra
}

func (scenario *scenarioUpgrade) SetVersions(versions ...string) {
	scenario.versions = versions
}

func (scenario *scenarioUpgrade) Run(t *testing.T) {
	t.Helper()

	install := scenarioInstall{
		name:                 scenario.name,
		manifestTemplatePath: scenario.manifestTemplatePath,
		infra:                scenario.infra,
		versions:             scenario.versions,
	}

	install.install(t)
	scenario.upgrade(t)
	scenario.test(t)
}

func (scenario *scenarioUpgrade) upgrade(t *testing.T) {
	// TODO: add upgrade logic
}

func (scenario *scenarioUpgrade) test(t *testing.T) {
	// TODO: add some testings
}

func (scenario *scenarioUpgrade) GenerateTests(wr io.Writer, generatorType GeneratorType, cfg ProwConfig) error {
	if len(scenario.versions) != 2 {
		return fmt.Errorf("expected only 2 versions")
	}

	type upgradeFromTo struct {
		From string
		To   string
	}

	up := upgradeFromTo{
		From: scenario.versions[0],
		To:   scenario.versions[1],
	}

	type templateData struct {
		Infra       string
		Scenario    string
		FromVersion string
		ToVersion   string
		TestTitle   string
	}

	var (
		data     []templateData
		prowJobs []ProwJob
	)

	testTitle := fmt.Sprintf("Test%s%sFrom%s_To%s",
		titleize(scenario.infra.name),
		scenario.Title(),
		titleize(up.From),
		titleize(up.To),
	)

	data = append(data, templateData{
		TestTitle:   testTitle,
		Infra:       scenario.infra.name,
		Scenario:    scenario.name,
		FromVersion: up.From,
		ToVersion:   up.To,
	})

	prowJobs = append(prowJobs,
		newProwJob(
			pullProwJobName(scenario.infra.name, scenario.name, "from", up.From, "to", up.To),
			scenario.infra.labels,
			testTitle,
			cfg,
		),
	)

	switch generatorType {
	case GeneratorTypeGo:
		tpl, err := template.New("").Parse(upgradeScenarioTemplate)
		if err != nil {
			return err
		}

		return tpl.Execute(wr, data)
	case GeneratorTypeYAML:
		buf, err := yaml.Marshal(prowJobs)
		if err != nil {
			return err
		}

		n, err := wr.Write(buf)
		if err != nil {
			return err
		}

		if n != len(buf) {
			return fmt.Errorf("wrong number of bytes written, expected %d, wrote %d", len(buf), n)
		}

		return nil
	}

	return fmt.Errorf("unknown generator type %d", generatorType)
}

const upgradeScenarioTemplate = `
{{- range . }}
func {{ .TestTitle }}(t *testing.T) {
	infra := Infrastructures["{{ .Infra }}"]
	scenario := Scenarios["{{ .Scenario }}"]
	scenario.SetInfra(infra)
	scenario.SetVersions("{{ .FromVersion }}", "{{ .ToVersion }}")
	scenario.Run(t)
}
{{ end -}}
`
