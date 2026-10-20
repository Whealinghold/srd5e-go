// Package contenttest provides test doubles and contract suites for the
// content ports, in the manner of net/http/httptest.
//
// [FakeSource] is an in-memory [content.Source] with scriptable failure and a
// record of the queries it saw. [RunSourceContract] and [RunReadOnlyContract]
// are the suites every [content.Source] adapter and double must pass.
//
// [FakeDecoder] is a [content.Decoder] double scripted per input document: a
// (Set, error) pair or a panic value, with a record of every input.
// [RunDecoderContract] is the suite every [content.Decoder] must pass.
//
// A consumer that writes its own Source or Decoder reuses the suites from its
// own tests: build the adapter in the factory, call the suite with *testing.T,
// and for the decoder supply one valid and one invalid document of its format:
//
//	func TestMyDecoder(t *testing.T) {
//		contenttest.RunDecoderContract(t,
//			func(*testing.T) content.Decoder { return myDecoder{} },
//			validDocument, invalidDocument)
//	}
//
// The decoder suite checks only what is true of every format (a valid document
// decodes, an invalid one is rejected with problems that match
// srderr.ErrInvalidInput, nothing panics, results repeat, the input is not
// retained); what a document means is the decoder's own tests. None of this is
// for production use.
package contenttest
