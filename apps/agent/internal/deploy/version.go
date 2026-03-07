package deploy

import "github.com/paasdeploy/shared/pkg/version"

func detectAppVersion(runtime, appDir string) string {
	return version.DetectAppVersion(runtime, appDir)
}
