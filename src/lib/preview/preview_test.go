package preview

const inI, resI string = "imgIn.jpg", "../../resI.jpg"
const inV, resV string = "vidIn.mp4", "../../vidIn.gif"

/*
func TestImageThumbGen(t *testing.T) {
	os.Remove(resI)

	f, e := os.Open("../../" + inI)
	if e != nil {
		t.Fatal(e, f)
	}
	inI, _, _ := fileutils.GenPreviewConvertPath(inI, "../../", "../../")
	cmd := getPrevCont()
	cmd.SetPaths(inI, resI, "image")
	g := getGen()
	g.Process(cmd)
	if _, err := os.Stat(resI); os.IsNotExist(err) {
		os.Remove(resI)
		t.Fatal("Thumbnail gen failed! path", resI)
	}
	os.Remove(resI)

}
func TestVideoThumbGen(t *testing.T) {
	os.Remove(resV)
	f, e := os.Open("../../" + inV)
	if e != nil {
		t.Fatal(e, f)
	}
	inV, _, _ := fileutils.GenPreviewConvertPath(inV, "../../", "../../")
	cmd := getPrevCont()
	cmd.SetPaths(inV, resV, "video")
	g := getGen()
	g.Process(cmd)
	if _, err := os.Stat(resV); os.IsNotExist(err) {
		os.Remove(resV)
		t.Fatal("Video gen failed! path", resV)
	}
	os.Remove(resV)
}*/

func getPrevCont() (res *PreviewData) {
	res = new(PreviewData)
	res.Setup(".", "../../convert.sh")

	return
}
func getGen() (res *PreviewGen) {
	res = new(PreviewGen)
	return
}
