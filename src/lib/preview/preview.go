package preview

import (
	"github.com/browsefile/backend/src/lib/fileutils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// should be 1 global object
type PreviewGen struct {
	ch           chan *PreviewData
	threadsCount int
}

func genPrew(pd *PreviewData) {
	if _, err := os.Stat(pd.out); os.IsNotExist(err) {
		cmd := exec.Command("/bin/sh", pd.convert, pd.in, pd.out, pd.fType)
		cmd.Dir = pd.dir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			log.Println(err)
		}
	}

}
func (p *PreviewGen) Setup(t int) {
	p.threadsCount = t
	if p.threadsCount <= 0 {
		p.threadsCount = 1
	} else {
		//todo make async
		p.ch = make(chan *PreviewData, 10000)
		for ; p.threadsCount > 0; p.threadsCount-- {
			go func() {
			Begin:
				genPrew(<-p.ch)
				goto Begin
			}()
		}
	}
}
func (p *PreviewGen) Process(pc *PreviewData) {
	if len(pc.in) == 0 || len(pc.out) == 0 {
		log.Printf("Error, in(%v) or out(%v) paths are empty ", pc.in, pc.out)
	} else if _, err := os.Stat(pc.out); err != nil {
		dirPath := filepath.Dir(pc.out)
		_, err := os.Stat(dirPath)
		if err != nil {
			err = os.MkdirAll(dirPath, 0775)
		}
		//do not wait for video preview generation, because it can take a while!
		if p.threadsCount == 1 && !strings.EqualFold(pc.fType, "video") {
			genPrew(pc)
			//otherwise run async, and cant be sure in return result
		} else {
			//run async, for immediate response
			p.ch <- pc

		}
	}
}

func (pd PreviewGen) GetDefaultData(in, out, t string) (rs *PreviewData) {
	rs = new(PreviewData)
	/*dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println("could not determinate current working folder")
		log.Fatal(err)
	}*/
	rs.Setup("./", "bfconvert.sh")
	if len(in) > 0 && len(out) > 0 && len(t) > 0 {
		rs.SetPaths(in, out, t)
	}

	return
}

//will generate previews recursively for scope
func (p *PreviewGen) ProcessPath(scope string, previewScope string) {
	err := filepath.Walk(scope,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			ok, t := fileutils.GetBasedOnExtensions(path)
			if ok && (strings.EqualFold("image", t) || strings.EqualFold("video", t)) {
				var out string
				out, err = fileutils.GenPreviewConvertPath(path, scope, previewScope)
				//yep generate in 1 thread, because in case n files, it can run out of ram on devices with low ram
				genPrew(p.GetDefaultData(path, out, t))
			}

			return nil
		})
	if err != nil {
		log.Println(err)
	}

}

type PreviewData struct {
	//paths to the shell scripts
	convert string
	//working dir of scripts
	dir string
	//paths for files
	in, out string
	//file type
	fType string
}

func (c *PreviewData) Setup(dir, convert string) {
	c.dir = dir
	c.convert = convert
}

func (c *PreviewData) SetPaths(in, out string, fType string) {
	c.in = in
	c.out = out
	c.fType = fType
}
