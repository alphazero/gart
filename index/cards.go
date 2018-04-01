// Doost!

package index

// An index.Card describes, in full, a gart object. Cards are stored as individual
// files in .gart/index/cards/ directory in a manner similar to git blobs. Cards
// are human readable, carriage return (\n) delimited files. (Given that typical
// host OS file system will allocate (typically) up to 4k per inode (even for a 1
// byte file) the design choice for plain-text (unicode) encoding of cards seems
// reasonable. Card files can also be compressed.
//
// Each card (redundantly) stores the object identity (Oid), associated tags (in
// plain text and not via a bitmap), systemic tags, system timestamps, and crc.
//
// Use cases for index.Card:
//
// Being the 'leaves' of the OS FS's (gratis) b-tree index structure, index.Cards
// allow for O(log) lookup of an object's details via Oid. Both individual file
// queries (i.e. gart-find -f <some-file>) and tag based queries (i.e. gart-find
// -tags <csv tag list>) internally resolve to one or more system.Oids. Given an
// Oid, access to the associated card is via os.Open.
//
// Cards are also the fundamental index recovery mechanism for gart. Given the
// set of index.Cards, the Object-index (object.idx), and associated Tag bitmaps
// can be rebuilt. Even the Tag Dictionary can be rebuilt given the set of Cards.
//
// For this reason, and given their small size, index.Cards are always read in full
// in RDONLY mode and updates are via SYNC'ed swaps.
//
// Card file format:
//
//    line  datum -- all lines are \n delimited.
//
//    0:    %016x formatted representation of CRC64 of card file lines 1->n.
//	  1:    %016x formatted object.idx key.
//    2:    0%16x %016x %d formatted create, update, and revision number.
//    3:    reserved for flags if any. this line may simply be a \n.
//    4:    %d %d formatted tag-count and initial line number for tags
//    5:    %d %d formatted path-count and initial line number for paths
//    6:    %d %d formatted systemics-count and initial line number for systemics
//    <path-count> lines are absolute path specs.
//    <tag-count> lines are tag names.
//    <systemic-count> lines are systemic attributes and flags.
//
// Example:
//
//  --- begin ----------------------
//  1:	 73cb3858a687a849
//  2:	 cd777f8ec7a2743f8190f54f5c189607357a29bd86fd49f006fef81647d99dbb
//  3:	 15210c6ca746f5ad 15210c6d48ad124f 1
//  4:	 7
//  5:	 Friend, Doost, Beloved, Salaam, Samad, Sultan, LOVE
//  6:	 2
//  7:	 .go, mar-31-2018
//  8:	 2
//  9:	 /Users/alphazero/Code/go/src/gart/index/ftest/test-index.go
//  10:	 /Volumes/OpenGate/Backups/ove/alphazero/Code/go/src/gart/index/ftest/test-index.go
//  --- end ------------------------
//
// REVU	each card load has to decode crc and timestamps. it also needs to decode
//		the counts. The rest are plain text in the binary encoded version as well.
//      The only difference, really, from 1.0 version is that we no longer use a BAH
//      and have no limits on number of tags. (before the BAH had to be 255 bytes
// 		max and of course since it was a bitmap, we needs tag.dict file to recover.)
//
//		One argument for plain-text is that it is 'human readable', but counter arg
//		is that there will be a gart-info -oid to decode it.
//
//		Parsing will not be faster or simpler. We still have to chase the \n terminal.
//
//		Parsing binary will be faster. It is true that it will be 'noise' for one
// 		card read, but still saving piping find . to gart-add will add up those
//		incremental +deltas.
//
//	    Reminder that the binary form will have <path-len><path-in-plaintext>, etc.
//		so parsing is deterministic.
//
// TODO sleep on this.
//
// REVU the simplest thing that would work:
//
//		a card simply is:
//		object type : in { blob, file }
//		list of paths for file objects
//		or
//		embedded blob
//
//		and that's it.
