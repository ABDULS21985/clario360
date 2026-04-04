package security_test

import (
	"testing"

	security "github.com/clario360/platform/internal/security"
)

// newSanitizer creates a fresh Sanitizer for each test.
func newSanitizer() *security.Sanitizer {
	return security.NewSanitizer()
}

// --- UNION SELECT variants ---

func TestSQLInjection_UnionSelectStar(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION SELECT * FROM users--")
	if err == nil {
		t.Fatal("expected UNION SELECT * FROM users-- to be detected as SQL injection")
	}
}

func TestSQLInjection_UnionAllSelect(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("1 UNION ALL SELECT 1,2,3")
	if err == nil {
		t.Fatal("expected UNION ALL SELECT 1,2,3 to be detected as SQL injection")
	}
}

func TestSQLInjection_UnionSelectWithWhitespace(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION  SELECT password FROM admins--")
	if err == nil {
		t.Fatal("expected UNION SELECT with extra whitespace to be detected")
	}
}

func TestSQLInjection_UnionSelectLowerCase(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' union select username, password from users--")
	if err == nil {
		t.Fatal("expected lowercase union select to be detected")
	}
}

// --- Piggyback attacks ---

func TestSQLInjection_PiggybackDropTable(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DROP TABLE users;--")
	if err == nil {
		t.Fatal("expected DROP TABLE to be detected as SQL injection")
	}
}

func TestSQLInjection_PiggybackDeleteFrom(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DELETE FROM sessions;--")
	if err == nil {
		t.Fatal("expected DELETE FROM to be detected as SQL injection")
	}
}

func TestSQLInjection_PiggybackUpdate(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; UPDATE users SET role='admin'--")
	if err == nil {
		t.Fatal("expected UPDATE SET to be detected as SQL injection")
	}
}

func TestSQLInjection_PiggybackInsert(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; INSERT INTO admin VALUES('hacker')--")
	if err == nil {
		t.Fatal("expected INSERT INTO to be detected as SQL injection")
	}
}

func TestSQLInjection_PiggybackDropDatabase(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DROP DATABASE production;--")
	if err == nil {
		t.Fatal("expected DROP DATABASE to be detected as SQL injection")
	}
}

// --- Tautology attacks ---

func TestSQLInjection_TautologyStringEquals(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' OR '1'='1")
	if err == nil {
		t.Fatal("expected string tautology ' OR '1'='1 to be detected")
	}
}

func TestSQLInjection_TautologyNumericEquals(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' OR 1=1--")
	if err == nil {
		t.Fatal("expected numeric tautology ' OR 1=1-- to be detected")
	}
}

func TestSQLInjection_TautologyAlwaysTrue(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' OR 'a'='a")
	if err == nil {
		t.Fatal("expected tautology ' OR 'a'='a to be detected")
	}
}

// --- Comment attacks ---

func TestSQLInjection_CommentTerminator(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("admin'--")
	if err == nil {
		t.Fatal("expected comment terminator admin'-- to be detected")
	}
}

func TestSQLInjection_BlockComment(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("test /* comment */")
	if err == nil {
		t.Fatal("expected block comment /* */ to be detected")
	}
}

func TestSQLInjection_InlineCommentBypass(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION/**/SELECT password FROM users--")
	if err == nil {
		t.Fatal("expected inline comment bypass to be detected")
	}
}

// --- Command execution ---

func TestSQLInjection_XPCmdShell(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; EXEC xp_cmdshell('whoami')--")
	if err == nil {
		t.Fatal("expected xp_cmdshell to be detected as SQL injection")
	}
}

func TestSQLInjection_ExecFunction(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; EXEC('SELECT 1')--")
	if err == nil {
		t.Fatal("expected EXEC() function to be detected")
	}
}

// --- Timing attacks ---

func TestSQLInjection_WaitforDelay(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; WAITFOR DELAY '0:0:5'--")
	if err == nil {
		t.Fatal("expected WAITFOR DELAY to be detected as SQL injection")
	}
}

func TestSQLInjection_SleepFunction(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; SELECT SLEEP(5)--")
	if err == nil {
		t.Fatal("expected SLEEP() to be detected as SQL injection")
	}
}

// --- File operations ---

func TestSQLInjection_IntoOutfile(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION SELECT * INTO OUTFILE '/tmp/dump'--")
	if err == nil {
		t.Fatal("expected INTO OUTFILE to be detected as SQL injection")
	}
}

func TestSQLInjection_LoadFile(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION SELECT LOAD_FILE('/etc/passwd')--")
	if err == nil {
		t.Fatal("expected LOAD_FILE to be detected as SQL injection")
	}
}

// --- Encoding attacks ---

func TestSQLInjection_BenchmarkTiming(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; BENCHMARK(1000000,SHA1('test'))--")
	if err == nil {
		t.Fatal("expected BENCHMARK() to be detected as SQL injection")
	}
}

func TestSQLInjection_CharEncoding(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("CHAR(115,101,108,101,99,116)")
	if err == nil {
		t.Fatal("expected CHAR() encoding to be detected as SQL injection")
	}
}

func TestSQLInjection_HexEncoding(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("0x61646d696e")
	if err == nil {
		t.Fatal("expected hex encoding 0x61646d696e to be detected as SQL injection")
	}
}

// --- Mixed case / obfuscation ---

func TestSQLInjection_MixedCaseUnion(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UnIoN sElEcT * FROM users--")
	if err == nil {
		t.Fatal("expected mixed-case UNION SELECT to be detected")
	}
}

func TestSQLInjection_MixedCaseDrop(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DrOp TaBlE users;--")
	if err == nil {
		t.Fatal("expected mixed-case DROP TABLE to be detected")
	}
}

func TestSQLInjection_TabSeparated(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("';\tDROP\tTABLE\tusers;--")
	if err == nil {
		t.Fatal("expected tab-separated DROP TABLE to be detected")
	}
}

func TestSQLInjection_NewlineSeparated(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DROP\nTABLE users;--")
	if err == nil {
		t.Fatal("expected newline-separated DROP TABLE to be detected")
	}
}

// --- Additional attack vectors ---

func TestSQLInjection_XPRegRead(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; EXEC xp_regread('HKLM','SOFTWARE')--")
	if err == nil {
		t.Fatal("expected xp_regread to be detected")
	}
}

func TestSQLInjection_IntoDumpfile(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' SELECT * INTO DUMPFILE '/tmp/data'--")
	if err == nil {
		t.Fatal("expected INTO DUMPFILE to be detected")
	}
}

func TestSQLInjection_SleepWithSpaces(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; SELECT SLEEP( 10 )--")
	if err == nil {
		t.Fatal("expected SLEEP with spaces to be detected")
	}
}

// --- Legitimate inputs that must pass ---

func TestSQLInjection_LegitimateNormalText(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Hello, this is a normal message.")
	if err != nil {
		t.Fatalf("legitimate text was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateEmail(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("user@example.com")
	if err != nil {
		t.Fatalf("email address was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateURL(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("https://example.com/path?query=value&other=123")
	if err != nil {
		t.Fatalf("URL was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateJSONString(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection(`{"name": "John", "role": "admin"}`)
	if err != nil {
		t.Fatalf("JSON string was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateSelectYourOption(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Select your option from the menu")
	if err != nil {
		t.Fatalf("natural language with 'Select' was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateDropDownMenu(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Drop down menu for categories")
	if err != nil {
		t.Fatalf("natural language with 'Drop' was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateDeleteButton(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Press the Delete button to remove the item")
	if err != nil {
		t.Fatalf("natural language with 'Delete' was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateNumbersAndSpecialChars(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Order #12345 - $99.99")
	if err != nil {
		t.Fatalf("order string was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateUnicodeText(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("Caf\u00e9 au lait, s'il vous pla\u00eet")
	if err != nil {
		t.Fatalf("unicode text was incorrectly flagged: %v", err)
	}
}

func TestSQLInjection_LegitimateEmptyString(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("")
	if err != nil {
		t.Fatalf("empty string was incorrectly flagged: %v", err)
	}
}

// --- InjectionError type assertion ---

func TestSQLInjection_ErrorIsInjectionError(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("' UNION SELECT * FROM users--")
	if err == nil {
		t.Fatal("expected an error")
	}
	injErr, ok := err.(*security.InjectionError)
	if !ok {
		t.Fatalf("expected *InjectionError, got %T", err)
	}
	if injErr.Type != "sql_injection" {
		t.Fatalf("expected Type 'sql_injection', got %q", injErr.Type)
	}
	if injErr.Category == "" {
		t.Fatal("expected non-empty Category")
	}
}

func TestSQLInjection_ErrorMessageFormat(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoSQLInjection("'; DROP TABLE users;--")
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

// --- ValidateIdentifier tests ---

func TestValidateIdentifier_ValidName(t *testing.T) {
	err := security.ValidateIdentifier("users_table")
	if err != nil {
		t.Fatalf("valid identifier was rejected: %v", err)
	}
}

func TestValidateIdentifier_RejectsDangerousReservedWord(t *testing.T) {
	err := security.ValidateIdentifier("DROP")
	if err == nil {
		t.Fatal("expected reserved word DROP to be rejected")
	}
}

func TestValidateIdentifier_RejectsSpecialChars(t *testing.T) {
	err := security.ValidateIdentifier("users;--")
	if err == nil {
		t.Fatal("expected identifier with special chars to be rejected")
	}
}
