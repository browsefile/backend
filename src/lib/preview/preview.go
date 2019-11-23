package preview

import (
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/lib/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// should be 1 global object
type PreviewGen struct {
	ch           chan *PreviewData
	threadsCount int
	scriptPath   string
}

func genPrew(pd *PreviewData) {
	if !utils.Exists(pd.out) {
		//create missed paths
		os.MkdirAll(filepath.Dir(pd.out), cnst.PERM_DEFAULT)
		cmd := exec.Command("/bin/sh", pd.convert, pd.in, pd.out, pd.fType)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			log.Println(err)
		}
	}

}
func (p *PreviewGen) Setup(t int, scr string) {
	p.threadsCount = t
	p.scriptPath = scr
	if p.threadsCount <= 0 {
		p.threadsCount = 1
	} else {
		//todo make async
		p.ch = make(chan *PreviewData, 10000)
		for ; p.threadsCount > 0; p.threadsCount-- {
			go func() {
			Begin:
				scp := <-p.ch
				_, t := utils.GetFileType(scp.in)
				genPrew(p.GetDefaultData(scp.in, scp.out, t))
				//p.ProcessPath(filepath.Dir(scp.in), filepath.Dir(scp.out))
				goto Begin
			}()
		}
	}
}
func (p *PreviewGen) Process(pc *PreviewData) {
	if len(pc.in) == 0 || len(pc.out) == 0 {
		log.Printf("Error, in(%v) or out(%v) paths are empty ", pc.in, pc.out)
	} else if _, err := os.Stat(pc.out); err != nil {
		p.ch <- pc
	}
}
func (pd PreviewGen) GetDefaultData(in, out, t string) (rs *PreviewData) {
	rs = new(PreviewData)
	rs.Setup(pd.scriptPath)
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

			if ok, t := utils.GetFileType(path); ok && (cnst.IMAGE == t || cnst.VIDEO == t) {
				genPrew(p.GetDefaultData(path, utils.GenPreviewConvertPath(path, scope, previewScope), t))
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
	//paths for files
	in, out string
	//file type
	fType string
}

func (c *PreviewData) Setup(convert string) {
	c.convert = convert
}

func (c *PreviewData) SetPaths(in, out string, fType string) {
	c.in = in
	c.out = out
	c.fType = fType
}
