// Doost!

// package bitmap defines compressible bitmaps with facilities for performing
// bitwise logical operations on the compressed bitmap.
//
// bitmap Bah0 is a variant of a byte-aligned-hybrid used for encoding of tags
// in a Card file. This form has byte aligned 'tile' and run-length encoded
// '0-fill' bytes. Not supporting '1-fill' blocks simplifies the code as Card
// bitmaps (unlike index tag-columns) are row-oriented and long sequences of
// '1' will not occur. Bah0 bitmaps also do not require explicit bit-wise ops
// as the typical use-case is to present a set of tag ids ([]ints) representing
// the bit position and check the Card's bah bitmap's bits at those positions.
//
// bitmap Wah is a canonical word-aligned-hybrid (per FastBit) and is used for
// the index's bitmap columns. The columns will have long bit sequences of '1'
// and will be of orders of magnitude longer length than the Card Bahs,
// reflecting the relative scale of the number of tags and the number of objects.
// Wah bitmaps also require direct compressed form bit-wise opertions.
//
// Both forms support the minimal bitmap.Bitmap interface to allow access to
// the compressed and uncompressed underlying byte array.
package bitmap
