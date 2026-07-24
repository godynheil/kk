// Copyright (C) 2026 Godynheil A. Quisto <godynheil@quisto.ph>
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import "testing"

func TestCodeLanguageDockerfile(t *testing.T) {
	lang, ok := CodeLanguage("Dockerfile")
	if !ok {
		t.Fatal("expected Dockerfile to be recognized")
	}
	if lang != "dockerfile" {
		t.Fatalf("unexpected language: %s", lang)
	}
}

func TestCodeLanguageExtensionsAndMetadata(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/main.go", true},
		{"go.mod", true},
		{"go.sum", true},
		{"go.work", true},
		{".gitmodules", true},
		{".gitignore", true},
		{".kkignore", true},
		{".env", true},
		{"LICENSE", true},
		{"README", true},
		{"Assets/MyShader.shader", true},
		{"Assets/CustomShader.hlsl", true},
		{"Assets/UnrealShader.usf", true},
		{"Assets/UnitySprite.png.meta", true},
		{"Assets/Assembly.asmdef", true},
		{"MyProject.uproject", true},
		{"Plugins/MyPlugin.uplugin", true},
		{"Assets/Sprite.png.import", true},
		{"Assets/video.mp4", false},
		{"Characters/model.fbx", false},
		{"Textures/Normal.uasset", false},
	}

	for _, tt := range tests {
		_, ok := CodeLanguage(tt.path)
		if ok != tt.want {
			t.Errorf("CodeLanguage(%q) = %v; want %v", tt.path, ok, tt.want)
		}
	}
}

func TestShouldTrackNonCodeFiles(t *testing.T) {
	tracks := Tracks{Patterns: []string{"*.fbx", "*.uasset"}}

	if !ShouldTrack("Characters/model.fbx", tracks) {
		t.Error("expected model.fbx to be tracked (matched pattern)")
	}

	if !ShouldTrack("Textures/Normal.uasset", tracks) {
		t.Error("expected Normal.uasset to be tracked (matched pattern)")
	}

	if ShouldTrack("src/main.go", tracks) {
		t.Error("expected main.go not to be tracked")
	}

	if ShouldTrack("TestAssets/untracked-test.bin", tracks) {
		t.Error("expected untracked .bin file not to be tracked when not in patterns")
	}
}

func TestShouldTrackDefaultsToAllNonCode(t *testing.T) {
	empty := Tracks{Patterns: []string{}}

	if !ShouldTrack("Characters/model.fbx", empty) {
		t.Error("expected model.fbx to be tracked by default (non-code)")
	}

	if !ShouldTrack("Textures/Normal.uasset", empty) {
		t.Error("expected Normal.uasset to be tracked by default (non-code)")
	}

	if !ShouldTrack("Assets/package.unitypackage", empty) {
		t.Error("expected .unitypackage to be tracked by default (non-code)")
	}

	if !ShouldTrack("Content/video.mp4", empty) {
		t.Error("expected .mp4 to be tracked by default (non-code)")
	}

	if !ShouldTrack("Models/hero.blend", empty) {
		t.Error("expected .blend to be tracked by default (non-code)")
	}

	if ShouldTrack("src/main.go", empty) {
		t.Error("expected main.go not to be tracked (code extension)")
	}

	if ShouldTrack("Assets/MyShader.shader", empty) {
		t.Error("expected .shader not to be tracked (code extension)")
	}

	if ShouldTrack("README", empty) {
		t.Error("expected README not to be tracked (code filename)")
	}

	if ShouldTrack(".gitignore", empty) {
		t.Error("expected .gitignore not to be tracked (dotfile)")
	}
}
