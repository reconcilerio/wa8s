/*
Copyright 2024.

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

package wit

import (
	"context"
	"regexp"

	"reconciler.io/wa8s/components"
)

func Extract(ctx context.Context, component []byte) (string, error) {
	return components.ExtractWIT(ctx, component)
}

var (
	imexRE = regexp.MustCompile(`(import|export)\s+([^;]+);`)
)

func ImportsExports(wit string) ([]string, []string) {
	// TODO consider using wit ast instead
	imports := []string{}
	exports := []string{}

	for _, match := range imexRE.FindAllStringSubmatch(wit, -1) {
		switch match[1] {
		case "import":
			imports = append(imports, match[2])
		case "export":
			exports = append(exports, match[2])
		}
	}

	return imports, exports
}
