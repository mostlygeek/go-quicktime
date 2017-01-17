package quicktime

import "io"
import "fmt"

// AtomArray is used to store the topmost level when building the atom tree ... there is no master top-level Atom in Quicktime.
type AtomArray []*Atom

// StringList stores a slice of Strings for BuildTreeConfig
type StringList []string

// BuildTreeConfig stores parameters for the BuildTree function.
type BuildTreeConfig struct {
	// List of Atom types which should be eager-loaded while building the tree.
	EagerloadTypes StringList
}

// Tests if a string occurs in a StringList.
func (list StringList) Includes( val string  ) bool {
	for _,str := range list {
		if str == val { return true }
	}
	return false
}


// BuildTree builds a tree of Atoms from an io.ReaderAt.   Rather than check for EOF, requires
// the io length to be pre-determined.   Takes a list of configuration closures, each of which
// is passed the BuildTreeConfig.
// Returns the top-level AtomArray.   On an error, this AtomArray will contain atoms up to the
// error.
func BuildTree(r io.ReaderAt, filesize int64, options ...func(*BuildTreeConfig) ) (AtomArray, error) {

	// Call configuration Functions
	config := BuildTreeConfig{}
	for _,opt := range options { opt(&config) }

	root := make([]*Atom, 0, 5)
	var err error = nil

	var offset int64 = 0
	for {
		fmt.Printf("Reading at %d\n", offset)
		atom, err := ReadAtomAt(r, offset)

		if err != nil {
			fmt.Println(err)
			return root, err
		}

		//  eagerload...
		if config.EagerloadTypes.Includes( atom.Type ) {
			fmt.Printf("Found atom %s, eagerloading...\n", atom.Type )
		 	atom.ReadData(r)
		}

		if atom.IsContainer() {
			if atom.HasData() {
				atom.BuildChildren()
			} else {
				atom.ReadChildren(r)
			}
		}

		offset += int64(atom.Size)
		root = append(root, &atom)

	}
	return root, err
}

// ReadChildren adds children to an Atom by reading from a ReaderAt.
func (atom *Atom) ReadChildren(r io.ReaderAt) {
	var offset int64 = atom.HeaderLength()
	for offset < int64(atom.Size) {
		loc := atom.Offset + offset
		//fmt.Println("Looking for header at:",loc)
		hdr, err := ReadAtomAt(r, loc)

		if err != nil {
			break
		}

		fmt.Printf("Found header at %d: %s\n", hdr,offset, hdr.Type)
		if hdr.IsContainer() {
			hdr.ReadChildren(r)
		}

		offset += int64(hdr.Size)

		atom.Children = append(atom.Children, &hdr)

	}
}

// BuildChildren adds children to an Atom after its data has been loaded.
// If the Atom already has children, behavior is undetermined.
func (atom *Atom) BuildChildren() {

	var offset int64 = 0
	for offset+atom.HeaderLength() < int64(atom.Size) {
		//fmt.Println("Looking for header at:", offset)
		hdr, err := ParseAtom(atom.Data[offset : offset+atom.HeaderLength()])

		if err == nil {
			//fmt.Println("Found header at", offset, ":", hdr.Type)
			hdr.Data = atom.Data[offset+atom.HeaderLength() : offset+int64(hdr.Size)]

			if hdr.IsContainer() {
				hdr.BuildChildren()
			}

			offset += int64(hdr.Size)

			atom.Children = append(atom.Children, &hdr)

		} else {
			fmt.Println("Error parsing atom:", err.Error())
			break
		}
	}
}
