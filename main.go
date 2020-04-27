package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/kardianos/osext"
	"github.com/tooolbox/archivex"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "gozak"
	app.Usage = ".mobi -> .azk"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "nocleanup, nc",
			Usage: "don't clean up temp files",
		},
	}
	app.Action = func(c *cli.Context) {
		if len(c.Args()) < 1 {
			log.Fatal("Must provide a path to the .mobi file")
		}
		targetPath := c.Args()[0]
		err := convertToAzk(targetPath, !c.Bool("nocleanup"))
		if err != nil {
			log.Fatalf("Gozak failed:\n%v", err)
		}
		log.Print("Gozak finished!")
	}
	app.Run(os.Args)
}

// This function will run azkcreator on the given file, and zip up the applicable output.
func convertToAzk(targetPath string, cleanup bool) error {

	dir := path.Dir(targetPath)

	// create the temp dir
	temp, err := ioutil.TempDir(dir, "temp")
	if err != nil {
		return err
	}

	// cleanup the temp dir
	defer func() {
		if cleanup {
			log.Print("Cleaning up...")
			os.RemoveAll(temp)
		} else {
			log.Print("Skipping cleanup...")
		}
	}()

	// call azkcreator
	wg := &sync.WaitGroup{}
	wg.Add(1)
	exdir, err := osext.ExecutableFolder()
	if err != nil {
		return fmt.Errorf("Error getting the directory of the executable: %v", err)
	}
	azkCmd := path.Join(exdir, "amazon", "azkcreator") + " --source " + targetPath + " --target " + temp
	log.Printf("executing: %s", azkCmd)
	out, err := exeCmd(azkCmd, wg)
	if err != nil {
		return fmt.Errorf("Error running azkcreator: (%v) output:\n%v", err, out)
	}
	wg.Wait()

	outputDir := path.Join(temp, "asin")

	// create an archive
	bare := strings.TrimSuffix(path.Base(targetPath), path.Ext(targetPath))
	archive := path.Join(dir, bare+".zip")

	// archive the applicable files using archivex
	zip := &archivex.ZipFile{}
	if err := zip.Create(archive); err != nil {
		return fmt.Errorf("archivex error: %v", err)
	}
	if err := zip.AddAll(outputDir, true); err != nil {
		return fmt.Errorf("archivex error: %v", err)
	}
	if err := zip.Close(); err != nil {
		return fmt.Errorf("archivex error: %v", err)
	}

	// the output from archivex has ".zip", so rename to ".azk"
	finalOutput := path.Join(dir, bare+".azk")
	if err := os.Rename(archive, finalOutput); err != nil {
		return err
	}

	log.Printf("Gozak created .azk file at: %v", finalOutput)

	return nil
}

// exeCmd executes a shell command.  Returns the stdout, and whether there was an error in trying to run the cmd.
// The caller of this function will need to determine if the cmd failed based on the output.
func exeCmd(cmd string, wg *sync.WaitGroup) (outPut string, err error) {

	// splitting head => g++ parts => rest of the command
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	// execute
	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		return string(out), err
	}

	// Wait for it to finish; signal the waitgroup that it's done
	wg.Done()
	return string(out), nil
}
