package internal

import (
	"reflect"
	"testing"
)

func TestChunks_ZeroSliceLenAndZeroChunkSize_Error(t *testing.T) {
	_, err := chunks(0, 0)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_OneSliceLenAndZeroChunkSize_Error(t *testing.T) {
	_, err := chunks(1, 0)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_PositiveSliceLenAndZeroChunkSize_Error(t *testing.T) {
	_, err := chunks(10, 0)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_ZeroSliceLenAndNegativeOneChunkSize_Error(t *testing.T) {
	_, err := chunks(0, -1)
	if err == nil {
		t.FailNow()
	}

}

func TestChunks_OneSliceLenAndNegativeOneChunkSize_Error(t *testing.T) {
	_, err := chunks(1, -1)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_PositiveSliceLenAndNegativeOneChunkSize_Error(t *testing.T) {
	_, err := chunks(10, -1)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_NegativeOneSliceLenAndZeroChunkSize_Error(t *testing.T) {
	_, err := chunks(-1, 0)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_NegativeOneSliceLenAndOneChunkSize_Error(t *testing.T) {
	_, err := chunks(-1, 1)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_NegativeOneSliceLenAndNegativeOneChunkSize_Error(t *testing.T) {
	_, err := chunks(-1, -1)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_NegativeSliceLenAndNegativeChunkSize_Error(t *testing.T) {
	_, err := chunks(-1, -10)
	if err == nil {
		t.FailNow()
	}
}

func TestChunks_ZeroSliceLenAndOneChunkSize_Empty(t *testing.T) {
	chunks, err := chunks(0, 1)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 0 {
		t.Fatal("bad length")
	}

}

func TestChunks_ZeroSliceLenAndPositiveChunkSize_Empty(t *testing.T) {
	chunks, err := chunks(0, 10)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 0 {
		t.Fatal("bad length")
	}

}

func TestChunks_OneSliceLenAndOneChunkSize_Chunk(t *testing.T) {
	chunks, err := chunks(1, 10)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 1 {
		t.Fatal("bad length")
	}
	if !reflect.DeepEqual(chunks, [][2]int{{0, 1}}) {
		t.Fatal("Not equals")
	}
}

func TestChunks_PositiveSliceLenAndEqualsChunkSize_Chunk(t *testing.T) {
	chunks, err := chunks(10, 10)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 1 {
		t.Fatal("bad length")
	}

	if !reflect.DeepEqual(chunks, [][2]int{{0, 10}}) {
		t.Fatal("Not equals")
	}
}

func TestChunks_TwiceSliceLenAndPositiveChunkSize_2Chunks(t *testing.T) {
	chunks, err := chunks(20, 10)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 2 {
		t.Fatal("bad length")
	}

	if !reflect.DeepEqual(chunks, [][2]int{{0, 10}, {10, 20}}) {
		t.Fatal("Not equals")
	}
}

func TestChunks_TwiceAndOneSliceLenAndPositiveChunkSize_3Chunks(t *testing.T) {
	chunks, err := chunks(21, 10)
	if err != nil {
		t.FailNow()
	}
	if len(chunks) != 3 {
		t.Fatal("bad length")
	}
	if !reflect.DeepEqual(chunks, [][2]int{{0, 10}, {10, 20}, {20, 21}}) {
		t.Fatal("Not equals")
	}
}
