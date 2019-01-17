package generators

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/Microsoft/fabrikate/core"
	"github.com/kyokomi/emoji"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func AddNamespaceToManifests(manifests string, namespace string) (namespacedManifests string, err error) {
	splitManifest := strings.Split(manifests, "\n---")

	for _, manifest := range splitManifest {
		parsedManifest := make(map[interface{}]interface{})
		yaml.Unmarshal([]byte(manifest), &parsedManifest)

		// strip any empty entries
		if len(parsedManifest) == 0 {
			continue
		}

		if parsedManifest["metadata"] != nil {
			metadataMap := parsedManifest["metadata"].(map[interface{}]interface{})
			metadataMap["namespace"] = namespace
		}

		updatedManifest, err := yaml.Marshal(&parsedManifest)
		if err != nil {
			return "", err
		}

		namespacedManifests += fmt.Sprintf("---\n%s\n", updatedManifest)
	}

	return namespacedManifests, nil
}

func MakeHelmRepoPath(component *core.Component) string {
	if len(component.Repo) == 0 {
		return component.PhysicalPath
	} else {
		return path.Join(component.PhysicalPath, "helm_repos", component.Name)
	}
}

func GenerateHelmComponent(component *core.Component) (manifest string, err error) {
	log.Println(emoji.Sprintf(":truck: generating component '%s' with helm with repo %s", component.Name, component.Repo))

	configYaml, err := yaml.Marshal(&component.Config.Config)
	if err != nil {
		log.Errorf("marshalling config yaml for helm generated component '%s' failed with: %s\n", component.Name, err.Error())
		return "", err
	}

	helmRepoPath := MakeHelmRepoPath(component)
	absHelmRepoPath, err := filepath.Abs(helmRepoPath)
	chartPath := path.Join(absHelmRepoPath, component.Path)
	absCustomValuesPath := path.Join(chartPath, "overriddenValues.yaml")

	log.Debugf("writing config %s to %s\n", configYaml, absCustomValuesPath)
	ioutil.WriteFile(absCustomValuesPath, configYaml, 0644)

	volumeMount := fmt.Sprintf("%s:/app/chart", chartPath)
	log.Debugf("templating with volumeMount: %s\n", volumeMount)

	name := component.Name
	if component.Config.Config["name"] != nil {
		name = component.Config.Config["name"].(string)
	}

	namespace := "default"
	if component.Config.Config["namespace"] != nil {
		namespace = component.Config.Config["namespace"].(string)
	}

	log.Debugf("templating with namespace: %s\n", namespace)

	output, err := exec.Command("docker", "run", "--rm", "-v", volumeMount, "alpine/helm:latest", "template", "/app/chart", "--values", "/app/chart/overriddenValues.yaml", "--name", name, "--namespace", namespace).Output()

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Errorf("helm template failed with: %s\n", ee.Stderr)
			return "", err
		}
	}

	stringManifests := string(output)

	// some helm templates expect install to inject namespace, so if namespace doesn't exist on resource manifests, manually inject it.
	if component.Config.Config["namespace"] != nil {
		stringManifests, err = AddNamespaceToManifests(stringManifests, component.Config.Config["namespace"].(string))
	}

	return stringManifests, err
}

func InstallHelmComponent(component *core.Component) (err error) {
	if len(component.Repo) == 0 {
		return nil
	}

	helmRepoPath := MakeHelmRepoPath(component)
	if err := exec.Command("rm", "-rf", helmRepoPath).Run(); err != nil {
		return err
	}

	if err := exec.Command("mkdir", "-p", helmRepoPath).Run(); err != nil {
		return err
	}

	log.Println(emoji.Sprintf(":helicopter: install helm repo %s for %s into %s", component.Repo, component.Name, helmRepoPath))
	return exec.Command("git", "clone", component.Repo, helmRepoPath, "--depth", "1").Run()
}
