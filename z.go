package xlsx

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/bitflip-software/xlsx/xmlprivate"
)

const (
	strContentTypes = "[Content_Types].xml"
	strRels         = "_rels/.rels"
	strWorkbookRels = "_rels/workbook.xml.rels"
)

const angle rune = '<'

// zinfo represents info about how to find the xlsx parts inside of the zip package
type zinfo struct {
	contentTypesIndex  int
	contentTypes       xmlprivate.ContentTypes
	relsIndex          int
	rels               xmlprivate.Rels
	wkbkName           string
	wkbkIndex          int
	wkbk               *zip.File
	wkbkRelsName       string
	wkbkRelsIndex      int
	wkbkRelsFile       *zip.File
	wkbkRels           xmlprivate.Rels
	sharedStringsIndex int
	sharedStringsName  string
	sharedStringsFile  *zip.File
	sharedStrings      sharedStrings
	sheetMeta          []sheetMeta
}

// zstruct represents the zip file reader and metadata about what was found in the xlsx package
type zstruct struct {
	r    *zip.Reader
	info zinfo
}

// zopen parses all of the necessary information from the xlsx package into a usable data structure
func zopen(zipData string) (z zstruct, err error) {
	b := []byte(zipData)
	brdr := bytes.NewReader(b)
	zr, err := zip.NewReader(brdr, int64(len(b)))

	if err != nil {
		return zstruct{}, err
	} else if zr == nil {
		return zstruct{}, errors.New("a nil zip.Reader was encountered")
	}

	z, err = zinit(zr)

	if err != nil {
		return zstruct{}, err
	}

	return z, nil
}

// zinit requires an open, error free *zip.Reader and returns a fully constructed zstruct
func zinit(zr *zip.Reader) (z zstruct, err error) {
	z.r = zr

	z.info, err = zparseContentTypes(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	z.info, err = zparseRels(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	z.info, err = zparseWorkbookLocation(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	z.info, err = zparseWorkbookRels(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	z.info, err = zparseSharedStrings(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	z.info, err = zparseSheetMetadata(zr, z.info)

	if err != nil {
		return zstruct{}, err
	}

	return z, nil
}

func zparseContentTypes(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	zi.contentTypesIndex = zfind(zr, strContentTypes)

	if zi.contentTypesIndex < 0 {
		return zi, err
	}

	file := zr.File[zi.contentTypesIndex]
	ctt := xmlprivate.ContentTypes{}
	ctbuf := bytes.Buffer{}
	ctwri := bufio.NewWriter(&ctbuf)
	ofile, err := file.Open()

	if err != nil {
		return zi, err
	} else {
		defer ofile.Close()
	}

	io.Copy(ctwri, ofile)
	err = xml.Unmarshal(ctbuf.Bytes(), &ctt)

	if err != nil {
		return zi, err
	}

	if len(ctt.Defaults) == 0 && len(ctt.Overrides) == 0 {
		return zi, fmt.Errorf("the %sstrings file has no contents", strContentTypes)
	}

	zi.contentTypes = ctt
	return zi, nil
}

// zparseWorkbookLocation must come after zparseRels
func zparseWorkbookLocation(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	// examples seen so far
	// http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	// http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument

	wkbkRelsIndex := -1

	for ix, rel := range zi.rels.Rels {
		rx, _ := regexp.Compile(`.+officeDocument.+officeDocument$`)
		match := rx.Match([]byte(rel.Type))
		if match {
			wkbkRelsIndex = ix
		}
	}

	if wkbkRelsIndex < 0 {
		for ix, rel := range zi.rels.Rels {
			rx, _ := regexp.Compile(`workbook\.xml$`)
			match := rx.Match([]byte(rel.Type))
			if match {
				wkbkRelsIndex = ix
			}
		}
	}

	if wkbkRelsIndex < 0 {
		return zi, nil
	}

	wkb := zi.rels.Rels[wkbkRelsIndex].Target
	zi.wkbkName = removeLeadingSlash(wkb)
	zi.wkbkIndex = zfind(zr, wkb)

	if zi.wkbkIndex < 0 {
		return zi, errors.New("the workbook could not be found")
	}

	zi.wkbk = zr.File[zi.wkbkIndex]
	return zi, nil
}

func zfind(zr *zip.Reader, filename string) (index int) {
	filename = removeLeadingSlash(filename)

	for ix, file := range zr.File {
		lcActual := strings.ToLower(removeLeadingSlash(file.FileHeader.Name))
		lcToFind := strings.ToLower(filename)
		lenActual := len(lcActual)
		lenToFind := len(lcToFind)

		if lenActual < lenToFind {
			continue
		}

		if lcActual[lenActual-lenToFind:] == lcToFind {
			return ix
		}
	}

	return -1
}

func zparseRels(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	zi.relsIndex = zfind(zr, strRels)

	if zi.contentTypesIndex < 0 {
		return zi, err
	}

	file := zr.File[zi.relsIndex]
	xstruct := xmlprivate.Rels{}
	fbuf := bytes.Buffer{}
	fwrite := bufio.NewWriter(&fbuf)
	ofile, err := file.Open()

	if err != nil {
		return zi, err
	} else {
		defer ofile.Close()
	}

	io.Copy(fwrite, ofile)
	err = xml.Unmarshal(fbuf.Bytes(), &xstruct)

	if err != nil {
		return zi, err
	}

	zi.rels = xstruct

	return zi, nil
}

// zparseWorkbookRels requires that the workbook has been found
func zparseWorkbookRels(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	wrelsName := wkbkRelsPath(zi.wkbkName)
	ix := zfind(zr, wrelsName)

	if ix < 0 {
		return zi, fmt.Errorf("workbook rels '%sstrings' could not be found", wrelsName)
	}

	zi.wkbkRelsIndex = ix
	zi.wkbkRelsFile = zr.File[ix]
	zi.wkbkRelsName = removeLeadingSlash(zi.wkbkRelsFile.Name)
	xstruct := xmlprivate.Rels{}
	fbuf := bytes.Buffer{}
	fwrite := bufio.NewWriter(&fbuf)
	ofile, err := zi.wkbkRelsFile.Open()

	if err != nil {
		return zi, err
	} else {
		defer ofile.Close()
	}

	io.Copy(fwrite, ofile)
	err = xml.Unmarshal(fbuf.Bytes(), &xstruct)

	if err != nil {
		return zi, err
	}

	zi.wkbkRels = xstruct

	return zi, nil
}

func zparseSharedStrings(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	// http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings
	index := -1

	for ix, rel := range zi.wkbkRels.Rels {
		rx, _ := regexp.Compile(`.+officeDocument.+sharedStrings$`)
		match := rx.Match([]byte(rel.Type))
		if match {
			index = ix
		}
	}

	if index < 0 {
		for ix, rel := range zi.wkbkRels.Rels {
			rx, _ := regexp.Compile(`sharedStrings\.xml$`)
			match := rx.Match([]byte(rel.Type))
			if match {
				index = ix
			}
		}
	}

	if index < 0 {
		// this is not an error, sharedStrings are not required
		return zi, nil
	}

	zi.sharedStringsName = joinWithWkbkPath(zi.wkbkName, zi.wkbkRels.Rels[index].Target)
	zi.sharedStringsIndex = zfind(zr, zi.sharedStringsName)

	if zi.sharedStringsIndex < 0 {
		// this is an error because we found a rels entry for sharedStrings but could not find the file
		return zi, fmt.Errorf("shared strings file '%sstrings' could not be found", zi.sharedStringsName)
	}

	zi.sharedStringsFile = zr.File[zi.sharedStringsIndex]
	zi.sharedStrings = newSharedStrings()

	fbuf := bytes.Buffer{}
	fwrite := bufio.NewWriter(&fbuf)
	ofile, err := zi.sharedStringsFile.Open()

	if err != nil {
		return zi, err
	} else {
		defer ofile.Close()
	}

	io.Copy(fwrite, ofile)
	strxml := string(fbuf.Bytes())
	runesxml := []rune(strxml)

	type stateST struct {
		inSI      bool
		inT       bool
		bufString bytes.Buffer
	}

	state := stateST{}
	l := len(runesxml)
	i := 0

outerloop:
	for i = 0; i < l; i++ {
		r := runesxml[i]

		if r == '<' {
			if state.inT {
				// get the string and put it into the shared strings
				shs := newSharedString()
				*shs.s = html.UnescapeString(state.bufString.String())
				zi.sharedStrings.add(shs)
				state.inSI = false
				state.inT = false
				state.bufString.Reset()
			} else if state.inSI {
				// check if we are getting a <t>
				if string(runesxml[i:mini(i+2, l)]) == "<t" {
					for ; i < l && runesxml[i] != '>'; i++ {
						// just advance the position
					}
					if runesxml[i] == '>' {
						state.inT = true
						continue outerloop
					}
				}
			} else {
				// check if we are getting a <si>
				peek := string(runesxml[i:mini(i+3, l)])
				if peek == "<si" {
					for ; i < l && runesxml[i] != '>'; i++ {
						// just advance the position
					}
					if runesxml[i] == '>' {
						state.inSI = true
						continue outerloop
					}
				}
			}
		} else if state.inT {
			state.bufString.WriteRune(r)
		}
	}

	if state.inT {
		// get the string and put it into the shared strings
		shs := newSharedString()
		*shs.s = html.UnescapeString(state.bufString.String())
		zi.sharedStrings.add(shs)
		state.inSI = false
		state.inT = false
		state.bufString.Reset()
	}

	return zi, nil
}

func zparseSheetMetadata(zr *zip.Reader, zi zinfo) (zout zinfo, err error) {
	// Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"

	zi.sheetMeta = make([]sheetMeta, 0)

	for _, rel := range zi.wkbkRels.Rels {
		rx, _ := regexp.Compile(`/worksheet$`)
		match := rx.Match([]byte(rel.Type))

		if !match {
			continue
		}

		relName := rel.Target
		sh := sheetMeta{}
		sh.name = joinWithWkbkPath(zi.wkbkName, relName)

		// TODO - find the sheet *zip.File
		// TODO - inspect the workbook to get the name of the sheet and the sheet index
		panic("this function isn't done being implemented")

		zi.sheetMeta = append(zi.sheetMeta, sh)
	}

	return zi, nil
}
