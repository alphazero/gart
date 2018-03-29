// Doost!

package index

// objects.idx file is a sequential list of object ids. The adjusted offset of
// the fixed witdth Oid data (32B) is the implicit 'key' for the object. The
// adjustment is accounting for the objects.idx header.
//
// On creation of new objects, an Oid entry is appended to the objects.idx file.
// The corresponding 'key' is recorded in the corresponding index.Card.
//
// On queries of objects for a given specification of tags (e.g AND or more
// selective logical expressions) an array of 'bits' is obtained from the Tagmap
// and these bit (positions) correspond to the 'keys' of object.idx, from which
// we maps the tagmap.bits -> object.keys -> Oids -> Cards.
