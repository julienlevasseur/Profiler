package profile

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/julienlevasseur/profiler/helpers"
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

var p, parseYamlResult, parseEnvrcResult map[string]string
var files, altFiles []string
var f bool

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
			configFile:                    "../../test/.profiler_cfg.yml",
			altConfigFile:                 "../../test/.alt_profiler_cfg.yml",
			profilesPath + ".env.yml":     "../../test/.env.yml",
			profilesPath + ".testrc":      "../../test/.testrc",
			profilesPath + ".test.yml":    "../../test/.test.yml",
			altProfilesPath + ".env.yml":  "../../test/.env.yml",
			altProfilesPath + ".testrc":   "../../test/.testrc",
			altProfilesPath + ".test.yml": "../../test/.test.yml",
		}

		// Copy the files needed to the tests:
		for dest, src := range filesToCopy {
			copyFile(src, dest)
		}

		files = helpers.ListFiles(profilesPath, ".*")
		p = GetProfile(profilesPath, "test")
		f = FileExist(configFile)
		parseYamlResult = ParseYaml("test/.test.yml")
		parseEnvrcResult = ParseEnvrc("test/.testrc")
	})

	Context("fileExist", func() {

		It("should be type bool", func() {
			Expect(reflect.TypeOf(f).Name()).To(Equal("bool"))
		})

		It("should return true", func() {
			Expect(f).To(BeTrue())
		})
	})

	Context("parseYaml", func() {

		It("should be type of map[string]string", func() {
			Expect(reflect.TypeOf(parseYamlResult).String()).To(Equal(
				"map[string]string",
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

	Context("parseEnvrc", func() {

		It("should be type of map[string]string", func() {
			Expect(reflect.TypeOf(parseEnvrcResult).String()).To(Equal(
				"map[string]string",
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

	Context("GetProfile", func() {

		It("should be type of map[string]string", func() {
			Expect(reflect.TypeOf(p).String()).To(Equal("map[string]string"))
		})

		It("should have specific key/value pair", func() {
			Expect(p).To(HaveKeyWithValue("key", "value"))
		})
	})

	AfterSuite(func() {
		// Cleanup:
		os.Remove(configFile)
		os.Remove(noCfgFilePath)
		os.RemoveAll(profilesPath)
	})
})
