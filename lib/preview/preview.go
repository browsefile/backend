package preview

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

// should be 1 global object
type PreviewGen struct {
	sMap  map[string]bool
	mutex *sync.Mutex
}

func (p *PreviewGen) Setup() {
	p.sMap = make(map[string]bool)
	p.mutex = new(sync.Mutex)
}

func (p *PreviewGen) ProcessSync(pc *PreviewData) {

	p.mutex.Lock()
	exists := p.sMap[pc.in]
	if !exists {
		p.sMap[pc.in] = true
	}
	p.mutex.Unlock()
	//prevent double run
	if exists {
		return
	}
	cmd := exec.Command("/bin/sh", pc.convert, pc.in, pc.out, pc.fType)
	cmd.Dir = pc.dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		log.Println(err)
		p.mutex.Lock()
		delete(p.sMap, pc.in)
		p.mutex.Unlock()

	} else {
		p.mutex.Lock()
		delete(p.sMap, pc.in)
		p.mutex.Unlock()
	}
}

func (pd PreviewGen) GetDefaultData() (rs *PreviewData) {

	rs = new(PreviewData)
	rs.Setup("./", "convert.sh")

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
