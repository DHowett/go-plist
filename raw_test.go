package plist

import (
	"bytes"
	"fmt"
)

func ExampleRawPlistValue_UnmarshalPlist() {
	type pbxprojFile struct {
		RootObjectRef string                    `plist:"rootObject"`
		Objects       map[string]*RawPlistValue `plist:"objects"`
	}

	type rootObject struct {
		IsA                       string `plist:"isa"`
		BuildConfigurationListRef string `plist:"buildConfigurationList"`
		CompatibilityVersion      string `plist:"compatibilityVersion"`
		Attributes                struct {
			LastUpgradeCheck int
		} `plist:"attributes"`
	}

	buf := bytes.NewReader([]byte(`// !$*UTF8*$!
{
  objects = {
    D015A98C1A9E25AC00A8721B /* Project object */ = {
      isa = PBXProject;
      attributes = {
        LastUpgradeCheck = 0610;
      };
      buildConfigurationList = D015A98F1A9E25AC00A8721B /* Build configuration list for PBXProject "Test" */;
      compatibilityVersion = "Xcode 3.2";
    };
  };
  rootObject = D015A98C1A9E25AC00A8721B /* Project object */;
}
`))

	var pbxproj pbxprojFile
	decoder := NewDecoder(buf)
	err := decoder.Decode(&pbxproj)
	if err != nil {
		fmt.Println(err)
	}

	var project rootObject
	err = decoder.DecodeElement(&project, pbxproj.Objects[pbxproj.RootObjectRef])
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(project)

	// Output: {PBXProject D015A98F1A9E25AC00A8721B Xcode 3.2 {610}}
}
