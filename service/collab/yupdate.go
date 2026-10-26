// Package collab relays and stores Yjs updates for real-time co-editing
// (web/src/lib/collab). Yjs runs in the browsers; the server never builds
// a document. It forwards updates between the clients of a room (one
// file), keeps them in an append-only log so a room can be rebuilt, and
// checks every message against the sender's project role.
package collab

// RootText is the name of the Y.Text holding a file's contents, and the
// key the web client reads (`doc.getText(RootText)`).
const RootText = "content"

// EncodeInsertText builds the Yjs (v1) update a fresh Y.Doc with the given
// client id produces for `getText(root).insert(0, text)`. The server seeds
// a room from the file on disk with it, so two browsers opening a new room
// at once never both insert the file (which would duplicate it).
//
// Layout: structs [numClients=1][numStructs=1][client][clock=0]
// [info=ContentString(4)][parent is a root key=1][root][text], then an
// empty delete set. An empty text is an empty update.
func EncodeInsertText(client uint64, root, text string) []byte {
	if text == "" {
		return []byte{0, 0}
	}
	var b []byte
	b = appendVarUint(b, 1)
	b = appendVarUint(b, 1)
	b = appendVarUint(b, client)
	b = appendVarUint(b, 0)
	b = append(b, 4)
	b = appendVarUint(b, 1)
	b = appendVarString(b, root)
	b = appendVarString(b, text)
	return appendVarUint(b, 0)
}

func appendVarUint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func appendVarString(b []byte, s string) []byte {
	b = appendVarUint(b, uint64(len(s)))
	return append(b, s...)
}
