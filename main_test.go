package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/julienlevasseur/profiler/cmd"
	"github.com/julienlevasseur/profiler/pkg/profile"
)

var configFile string = "/tmp/.profiler_cfg.yml"
var altConfigFile string = "/tmp/.alt_profiler_cfg.yml"
var profilesPath string = "/tmp/.profiler/"
var altProfilesPath string = "/tmp/.alt_profiler/"
var noCfgFilePath string = "/tmp/profiler_no_cfg.yml"

func createFolder(path string) {
	fmt.Println("Create temp folder " + path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func copyFile(source string, destination string) {
	input, err := ioutil.ReadFile(source)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(destination, input, 0644)
	if err != nil {
		fmt.Println("Error creating", destination)
		fmt.Println(err)
		return
	}
}

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Profiler")
}

var cfg, p profile.KeyValueMap
var files, altFiles []string
var f bool
var parseYamlResult, parseEnvrcResult profile.KeyValueMap

var _ = Describe("Profiler", func() {

	BeforeSuite(func() {
		// Create the temp folders for test:
		createFolder(profilesPath)
		createFolder(altProfilesPath)

		// This map represent the files to copy.
		// The "destination" is the key while the value is the "source"
		// to copy the files. Simply because a map can't contain duplicate keys and
		// some files have to be copied to 2 different destinations.
		filesToCopy := map[string]string{
			configFile:                    "test/.profiler_cfg.yml",
			altConfigFile:                 "test/.alt_profiler_cfg.yml",
			profilesPath + ".env.yml":     "test/.env.yml",
			profilesPath + ".testrc":      "test/.testrc",
			profilesPath + ".test.yml":    "test/.test.yml",
			altProfilesPath + ".env.yml":  "test/.env.yml",
			altProfilesPath + ".testrc":   "test/.testrc",
			altProfilesPath + ".test.yml": "test/.test.yml",
		}

		// Copy the files needed to the tests:
		for dest, src := range filesToCopy {
			copyFile(src, dest)
		}

		files = profile.ListFiles(profilesPath, ".*")
		p = profile.GetProfile(profilesPath, "test")
		f = profile.FileExist(configFile)
		parseYamlResult = profile.ParseYaml("test/.test.yml")
		parseEnvrcResult = profile.ParseEnvrc("test/.testrc")
		altFiles = profile.ListFiles(altProfilesPath, ".*")
	})

	Context("ListFiles", func() {

		It("should be type of []string", func() {
			Expect(reflect.TypeOf(files).String()).To(Equal("[]string"))
		})

		It("should contain 3 elements", func() {
			Expect(files).To(HaveLen(3))
		})

		It("should contain .env.yml, .testrc & .test.yml", func() {
			Expect(files).To(ContainElement(profilesPath + ".env.yml"))
			Expect(files).To(ContainElement(profilesPath + ".testrc"))
			Expect(files).To(ContainElement(profilesPath + ".test.yml"))
		})
	})

	Context("GetProfile", func() {

		It("should be type of KeyValueMap", func() {
			Expect(reflect.TypeOf(p).Name()).To(Equal("KeyValueMap"))
		})

		It("should have specific key/value pair", func() {
			Expect(p).To(
				HaveKeyWithValue(
					"key",
					"value",
				),
			)
		})
	})

	Context("FileExist", func() {

		It("should be type bool", func() {
			Expect(reflect.TypeOf(f).Name()).To(Equal("bool"))
		})

		It("should return true", func() {
			Expect(f).To(BeTrue())
		})
	})

	Context("ParseYaml", func() {

		It("should be type of KeyValueMap", func() {
			Expect(reflect.TypeOf(parseYamlResult).Name()).To(Equal(
				"KeyValueMap",
			))
		})

		It("should have specific key/value pair", func() {
			Expect(parseYamlResult).To(
				HaveKeyWithValue(
					"key",
					"value",
				),
			)
		})

	})

	Context("ParseEnvrc", func() {

		It("should be type of KeyValueMap", func() {
			Expect(reflect.TypeOf(parseEnvrcResult).Name()).To(Equal(
				"KeyValueMap",
			))
		})

		It("should have specific key/value pair", func() {
			Expect(parseEnvrcResult).To(
				HaveKeyWithValue(
					"key",
					"value",
				),
			)
		})

	})

	Context("Alternate config", func() {
		// Simulate a user setings his custom configFile path:
		os.Setenv("PROFILER_CFG", altConfigFile)
		
		It("should be type of []string", func() {
			Expect(reflect.TypeOf(altFiles).String()).To(Equal("[]string"))
		})
		
		It("should contain 3 elements", func() {
			Expect(altFiles).To(HaveLen(3))
		})
		
		It("should contain .env.yml, .testrc & .test.yml", func() {
			Expect(altFiles).To(ContainElement(altProfilesPath + ".test.yml"))
		})
	})
	
	Context("setConfigFile", func() {
		// Simulate a user setings his custom configFile path:
		os.Setenv("PROFILER_CFG", noCfgFilePath)

		cmd.InitConfig()

		It("should have created a config file", func() {
			Expect(noCfgFilePath).To(BeAnExistingFile())
		})
	})

	AfterSuite(func() {
		// Cleanup:
		os.Remove(configFile)
		os.Remove(noCfgFilePath)
		os.RemoveAll(profilesPath)
	})
})
