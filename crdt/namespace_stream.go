package crdt

import (
	"sort"

	"github.com/johnny-morrice/godless/crypto"
	"github.com/pkg/errors"
)

type StreamPoint struct {
	Text      PointText
	Signature crypto.SignatureText
}

func (point StreamPoint) Equals(other StreamPoint) bool {
	return point.Text == other.Text && point.Signature == other.Signature
}

func (point StreamPoint) Less(other StreamPoint) bool {
	if point.Text < other.Text {
		return true
	} else if point.Text > other.Text {
		return false
	}

	return point.Signature < other.Signature
}

type InvalidNamespaceEntry NamespaceStreamEntry

// FIXME not really a stream, whole is kept in memory.
type NamespaceStreamEntry struct {
	Table TableName
	Row   RowName
	Entry EntryName
	Point StreamPoint
}

func (entry NamespaceStreamEntry) samePoint(other NamespaceStreamEntry) bool {
	ok := entry.Table == other.Table
	ok = ok && entry.Row == other.Row
	ok = ok && entry.Entry == other.Entry
	ok = ok && entry.Point.Text == other.Point.Text
	return ok
}

func SortNamespaceStream(stream []NamespaceStreamEntry) {
	sort.Sort(byNamespaceStreamOrder(stream))
}

type byNamespaceStreamOrder []NamespaceStreamEntry

func (stream byNamespaceStreamOrder) Len() int {
	return len(stream)
}

func (stream byNamespaceStreamOrder) Swap(i, j int) {
	stream[i], stream[j] = stream[j], stream[i]
}

func (stream byNamespaceStreamOrder) Less(i, j int) bool {
	a, b := stream[i], stream[j]

	if a.Table < b.Table {
		return true
	} else if a.Table > b.Table {
		return false
	}

	if a.Row < b.Row {
		return true
	} else if a.Row > b.Row {
		return false
	}

	if a.Entry < b.Entry {
		return true
	} else if a.Entry > b.Entry {
		return false
	}

	return a.Point.Less(b.Point)
}

type streamBuilder struct {
	stream  []NamespaceStreamEntry
	invalid []InvalidNamespaceEntry
}

func (builder *streamBuilder) uniqueOrder() {
	sort.Sort(byNamespaceStreamOrder(builder.stream))
	builder.uniqSorted()
}

func (builder *streamBuilder) uniqSorted() {
	if len(builder.stream) < 2 {
		return
	}

	uniqIndex := 0
	for i := 1; i < len(builder.stream); i++ {
		entry := builder.stream[i]
		last := builder.stream[uniqIndex]

		if entry != last {
			uniqIndex++
			builder.stream[uniqIndex] = entry
		}
	}

	builder.stream = builder.stream[:uniqIndex+1]
}

func (builder *streamBuilder) makeStreamPoints(proto NamespaceStreamEntry, point Point) {
	if len(point.Signatures()) == 0 {
		entry := proto
		entry.Point = StreamPoint{Text: point.Text()}
		builder.stream = append(builder.stream, entry)
	}

	for _, sig := range point.Signatures() {
		entry := proto
		streamPoint, err := MakeStreamPoint(point.Text(), sig)

		if err != nil {
			entry.Point.Text = point.Text()
			builder.invalid = append(builder.invalid, InvalidNamespaceEntry(entry))
			continue
		}

		entry.Point = streamPoint
		builder.stream = append(builder.stream, entry)
	}
}

func MakeStreamPoint(text PointText, sig crypto.Signature) (StreamPoint, error) {
	sigText, err := crypto.PrintSignature(sig)

	if err != nil {
		return StreamPoint{}, err
	}

	return StreamPoint{Text: text, Signature: sigText}, nil
}

func readStreamPoint(stream []NamespaceStreamEntry) (Point, []InvalidNamespaceEntry, error) {
	const failMsg = "readStreamPoint failed"

	if len(stream) == 0 {
		return Point{}, nil, nil
	}

	first := stream[0]
	signatures := make([]crypto.Signature, 0, len(stream))

	var invalid []InvalidNamespaceEntry

	for _, entry := range stream {
		if !entry.samePoint(first) {
			notSame := errors.New("Corrupt stream")
			return Point{}, nil, errors.Wrap(notSame, failMsg)
		}

		if crypto.IsNilSignature(entry.Point.Signature) {
			continue
		}

		sig, err := crypto.ParseSignature(entry.Point.Signature)

		if err != nil {
			invalid = append(invalid, InvalidNamespaceEntry(entry))
			continue
		}

		signatures = append(signatures, sig)
	}

	point := PresignedPoint(first.Point.Text, signatures)

	return point, invalid, nil
}

func MakeTableStream(tableKey TableName, table Table) ([]NamespaceStreamEntry, []InvalidNamespaceEntry) {
	subNamespace := MakeNamespace(map[TableName]Table{
		tableKey: table,
	})
	return MakeNamespaceStream(subNamespace)
}

func MakeRowStream(tableKey TableName, rowKey RowName, row Row) ([]NamespaceStreamEntry, []InvalidNamespaceEntry) {
	table := MakeTable(map[RowName]Row{
		rowKey: row,
	})
	return MakeTableStream(tableKey, table)
}

func MakeNamespaceStream(ns Namespace) ([]NamespaceStreamEntry, []InvalidNamespaceEntry) {
	count := streamLength(ns)

	builder := &streamBuilder{stream: make([]NamespaceStreamEntry, 0, count)}

	ns.ForeachEntry(func(t TableName, r RowName, e EntryName, entry Entry) {
		proto := NamespaceStreamEntry{
			Table: t,
			Row:   r,
			Entry: e,
		}

		for _, point := range entry.GetValues() {
			builder.makeStreamPoints(proto, point)
		}
	})

	builder.uniqueOrder()

	return builder.stream, builder.invalid
}

func streamLength(ns Namespace) int {
	count := 0

	ns.ForeachEntry(func(t TableName, r RowName, e EntryName, entry Entry) {
		for _, point := range entry.GetValues() {
			sigCount := len(point.Signatures())
			if sigCount > 0 {
				count += sigCount
			} else {
				count++
			}
		}
	})

	return count
}

func ReadNamespaceStream(stream []NamespaceStreamEntry) (Namespace, []InvalidNamespaceEntry) {
	const failMsg = "ReadNamespaceStream failed"

	ns := EmptyNamespace()
	var invalidEntries []InvalidNamespaceEntry

	batchStart := 0
	for batchEnd := 1; batchEnd <= len(stream); batchEnd++ {
		startEntry := stream[batchStart]

		writePoint := false

		if batchEnd < len(stream) {
			entry := stream[batchEnd]
			writePoint = !entry.samePoint(startEntry)
		} else {
			writePoint = true
		}

		if writePoint {
			if batchEnd-batchStart == 1 {
				err := ns.addStreamEntry(startEntry)

				if err != nil {
					panic("BUG in namespace stream batch")
				}

				batchStart = batchEnd
				continue
			}

			batch := stream[batchStart:batchEnd]

			invalid, err := ns.addPointBatch(batch)

			invalidEntries = append(invalidEntries, invalid...)

			if err != nil {
				panic("BUG in namespace stream batch")
			}

			batchStart = batchEnd
		}

	}

	return ns, invalidEntries
}
