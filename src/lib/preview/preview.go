package preview

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
		p.ch = make(chan *PreviewData, 100*p.threadsCount)
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
		fmt.Printf("Error, in(%s) or out(%s) paths are empty ", pc.in, pc.out)
		fmt.Println("Error, in or out paths are empty ")
	} else if _, err := os.Stat(pc.out); err != nil {

		dirPath := filepath.Dir(pc.out)
		_, err := os.Stat(dirPath)
		if err != nil {
			err = os.MkdirAll(dirPath, 0775)
		}

		if p.threadsCount == 1 {
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
	rs.Setup("./", "convert.sh")
	if len(in) > 0 && len(out) > 0 && len(t) > 0 {
		rs.SetPaths(in, out, t)
	}

	return
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

func (c *PreviewData) SetPaths(inArr, outArr string, fType string) {
	c.in = inArr
	c.out = outArr
	c.fType = fType
}
