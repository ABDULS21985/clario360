package security_test

import (
	"testing"

	security "github.com/clario360/platform/internal/security"
)

// --- Script tag attacks ---

func TestXSS_BasicScriptTag(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<script>alert('XSS')</script>")
	if err == nil {
		t.Fatal("expected <script>alert('XSS')</script> to be detected as XSS")
	}
}

func TestXSS_ScriptSrcRemote(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<SCRIPT SRC=//evil.com/xss.js></SCRIPT>")
	if err == nil {
		t.Fatal("expected remote script src to be detected as XSS")
	}
}

func TestXSS_ScriptWithSpaces(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("< script >alert(1)</ script >")
	if err == nil {
		t.Fatal("expected script tag with spaces to be detected as XSS")
	}
}

func TestXSS_ScriptUpperCase(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<SCRIPT>document.cookie</SCRIPT>")
	if err == nil {
		t.Fatal("expected uppercase SCRIPT to be detected as XSS")
	}
}

// --- Event handler attacks ---

func TestXSS_ImgOnerror(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<img onerror=alert(1) src=x>")
	if err == nil {
		t.Fatal("expected img onerror to be detected as XSS")
	}
}

func TestXSS_BodyOnload(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<body onload=alert('XSS')>")
	if err == nil {
		t.Fatal("expected body onload to be detected as XSS")
	}
}

func TestXSS_DivOnmouseover(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<div onmouseover="alert('XSS')">hover me</div>`)
	if err == nil {
		t.Fatal("expected div onmouseover to be detected as XSS")
	}
}

func TestXSS_InputOnfocus(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<input onfocus=alert(1) autofocus>`)
	if err == nil {
		t.Fatal("expected input onfocus to be detected as XSS")
	}
}

// --- JavaScript URI attacks ---

func TestXSS_JavaScriptURI(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("javascript:alert(1)")
	if err == nil {
		t.Fatal("expected javascript: URI to be detected as XSS")
	}
}

func TestXSS_AnchorJavaScriptHref(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<a href="javascript:alert(1)">click</a>`)
	if err == nil {
		t.Fatal("expected anchor with javascript: href to be detected as XSS")
	}
}

func TestXSS_JavaScriptURIWithSpaces(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("javascript :alert(1)")
	if err == nil {
		t.Fatal("expected javascript URI with space to be detected as XSS")
	}
}

// --- Data URI attacks ---

func TestXSS_DataURIScript(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<img src="data:text/html,<script>alert(1)</script>">`)
	if err == nil {
		t.Fatal("expected data:text/html URI to be detected as XSS")
	}
}

// --- SVG attacks ---

func TestXSS_SVGOnload(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<svg onload=alert(1)>")
	if err == nil {
		t.Fatal("expected svg onload to be detected as XSS")
	}
}

func TestXSS_SVGOnloadSlash(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<svg/onload=alert(1)>")
	if err == nil {
		t.Fatal("expected svg/onload to be detected as XSS")
	}
}

// --- Iframe attacks ---

func TestXSS_IframeSrc(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<iframe src="evil.com"></iframe>`)
	if err == nil {
		t.Fatal("expected iframe to be detected as XSS")
	}
}

func TestXSS_IframeNoSrc(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<iframe srcdoc="<script>alert(1)</script>"></iframe>`)
	if err == nil {
		t.Fatal("expected iframe with srcdoc to be detected as XSS")
	}
}

// --- Object/Embed attacks ---

func TestXSS_ObjectData(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<object data="evil.swf">`)
	if err == nil {
		t.Fatal("expected object tag to be detected as XSS")
	}
}

func TestXSS_EmbedSrc(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<embed src="evil.swf">`)
	if err == nil {
		t.Fatal("expected embed tag to be detected as XSS")
	}
}

// --- Base tag hijacking ---

func TestXSS_BaseTagHijack(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<base href="https://evil.com/">`)
	if err == nil {
		t.Fatal("expected base tag to be detected as XSS")
	}
}

// --- CSS expression attacks ---

func TestXSS_CSSExpression(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`<div style="background:expression(alert(1))">`)
	if err == nil {
		t.Fatal("expected CSS expression to be detected as XSS")
	}
}

func TestXSS_CSSExpressionInStyle(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS(`expression(document.cookie)`)
	if err == nil {
		t.Fatal("expected standalone CSS expression() to be detected as XSS")
	}
}

// --- Template injection ---

func TestXSS_TemplateInjection(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("{{constructor.constructor('alert(1)')()}}")
	if err == nil {
		t.Fatal("expected template injection {{...}} to be detected as XSS")
	}
}

func TestXSS_AngularTemplateInjection(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("{{7*7}}")
	if err == nil {
		t.Fatal("expected Angular template injection to be detected as XSS")
	}
}

// --- Encoded payloads ---

func TestXSS_URLEncodedScript(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("%3Cscript%3Ealert(1)%3C/script%3E")
	if err == nil {
		t.Fatal("expected URL-encoded script to be detected as XSS")
	}
}

func TestXSS_URLEncodedEventHandler(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("%3Cimg%20onerror%3Dalert(1)%20src%3Dx%3E")
	if err == nil {
		t.Fatal("expected URL-encoded event handler to be detected as XSS")
	}
}

// --- Polyglot XSS ---

func TestXSS_PolyglotVector(t *testing.T) {
	s := newSanitizer()
	payload := "jaVasCript:/*-/*`/*'/*\"/**/(/* */oNcliCk=alert() )//%0D%0A"
	err := s.ValidateNoXSS(payload)
	if err == nil {
		t.Fatal("expected polyglot XSS vector to be detected")
	}
}

// --- Error type assertion ---

func TestXSS_ErrorIsInjectionError(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("<script>alert(1)</script>")
	if err == nil {
		t.Fatal("expected an error")
	}
	injErr, ok := err.(*security.InjectionError)
	if !ok {
		t.Fatalf("expected *InjectionError, got %T", err)
	}
	if injErr.Type != "xss" {
		t.Fatalf("expected Type 'xss', got %q", injErr.Type)
	}
	if injErr.Category == "" {
		t.Fatal("expected non-empty Category on XSS error")
	}
}

// --- Legitimate inputs that must pass ---

func TestXSS_LegitimateNormalText(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("This is a perfectly normal paragraph of text.")
	if err != nil {
		t.Fatalf("normal text was incorrectly flagged as XSS: %v", err)
	}
}

func TestXSS_LegitimateMathExpression(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("5 > 3 and 2 < 10")
	if err != nil {
		t.Fatalf("math expression '5 > 3' was incorrectly flagged as XSS: %v", err)
	}
}

func TestXSS_LegitimateURL(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("https://example.com/page?q=search&lang=en")
	if err != nil {
		t.Fatalf("URL was incorrectly flagged as XSS: %v", err)
	}
}

func TestXSS_LegitimateEmptyString(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("")
	if err != nil {
		t.Fatalf("empty string was incorrectly flagged as XSS: %v", err)
	}
}

func TestXSS_LegitimateCodeDiscussion(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("Use the function getData() to retrieve records")
	if err != nil {
		t.Fatalf("code discussion text was incorrectly flagged as XSS: %v", err)
	}
}

func TestXSS_LegitimatePlainParentheses(t *testing.T) {
	s := newSanitizer()
	err := s.ValidateNoXSS("The result (42) was expected")
	if err != nil {
		t.Fatalf("parentheses in text were incorrectly flagged as XSS: %v", err)
	}
}
