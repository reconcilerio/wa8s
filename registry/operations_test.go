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

package registry

// func TestPullAndPush(t *testing.T) {
// 	ctx := context.Background()

// 	ref, err := ResolveDigest(ctx, "scothis/logger")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	ctx = StashCache(context.Background(), "testdata/pull")
// 	component, pullConfig, err := Pull(ctx, ref)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	_ = os.WriteFile("testdata/pull.wasm", component, 0700)

// 	ctx = StashCache(context.Background(), "testdata/push")
// 	target, _ := name.ParseReference("localhost:5001/components/logging")
// 	_, pushConfig, err := Push(ctx, target, component)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if diff := cmp.Diff(pullConfig, pushConfig); diff != "" {
// 		t.Errorf("config (+pull, -push): \n%s", diff)
// 	}
// }
