package connector

import "testing"

func TestHDFSFormatDetection(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		head     []byte
		want     string
	}{
		{name: "ParquetMagic", filePath: "dataset.bin", head: []byte("PAR1data"), want: "parquet"},
		{name: "ORCMagic", filePath: "dataset.bin", head: []byte("ORCdata"), want: "orc"},
		{name: "CSVBySuffix", filePath: "customers.csv", head: []byte("id,email\n1,a@b.com\n"), want: "csv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectHDFSFormat(tt.filePath, tt.head); got != tt.want {
				t.Fatalf("detectHDFSFormat(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestHDFSPIIDetectionFromValues(t *testing.T) {
	_, emailFindings := scanDelimitedBuffer([]byte("email\nalice@example.com\nbob@example.com\n"), false)
	if len(emailFindings) == 0 || emailFindings[0].PIIType != "email" {
		t.Fatalf("email findings = %+v, want email", emailFindings)
	}

	_, cardFindings := scanDelimitedBuffer([]byte("card\n4242424242424242\n"), false)
	if len(cardFindings) == 0 || cardFindings[0].PIIType != "credit_card" {
		t.Fatalf("card findings = %+v, want credit_card", cardFindings)
	}

	_, ssnFindings := scanDelimitedBuffer([]byte("ssn\n123-45-6789\n"), false)
	if len(ssnFindings) == 0 || ssnFindings[0].PIIType != "national_id" {
		t.Fatalf("ssn findings = %+v, want national_id", ssnFindings)
	}
}

func TestHDFSSkipMarkerFiles(t *testing.T) {
	if !isMarkerFile("_SUCCESS") {
		t.Fatal("expected _SUCCESS to be treated as marker file")
	}
	if !isMarkerFile("part-0000.crc") {
		t.Fatal("expected .crc file to be treated as marker file")
	}
	if isMarkerFile("customers.csv") {
		t.Fatal("expected regular data file not to be treated as marker file")
	}
}
