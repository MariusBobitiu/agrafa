package email

import (
	"strings"
	"testing"
	"time"
)

func floatPtr(value float64) *float64 { return &value }
func intPtr(value int) *int           { return &value }
func int64Ptr(value int64) *int64     { return &value }

func baseAlertData() AlertTemplateData {
	resolved := time.Date(2026, 8, 29, 8, 9, 57, 0, time.UTC)
	return AlertTemplateData{
		ProjectID: 7, ProjectName: "Test", RuleType: "service_unhealthy", RuleLabel: "Service unhealthy",
		Severity: "critical", Status: "active", ResourceType: "service", ResourceName: "Landing", ServiceName: "Landing",
		ServiceCheckType: "HTTP", ServiceTarget: "https://landing.example.com/health", StatusCode: intPtr(503), ResponseTimeMs: int64Ptr(412), FailureReason: "Service Unavailable",
		AlertTitle: "⚠ Landing is unhealthy", AlertMessage: "HTTP check to https://landing.example.com/health returned 503 Service Unavailable.",
		TriggeredAt: time.Date(2026, 8, 29, 7, 49, 26, 0, time.UTC), ResolvedAt: &resolved, Duration: "20m 31s",
		ResourceURL: "https://app.agrafa.test/services/13", AlertsURL: "https://app.agrafa.test/alerts", NotificationsURL: "https://app.agrafa.test/settings?tab=notifications",
	}
}

func startTagWithMarker(t *testing.T, html, marker string) string {
	t.Helper()
	markerIndex := strings.Index(html, marker)
	if markerIndex < 0 {
		t.Fatalf("rendered HTML missing marker %q", marker)
	}
	start := strings.LastIndex(html[:markerIndex], "<")
	endOffset := strings.Index(html[markerIndex:], ">")
	if start < 0 || endOffset < 0 {
		t.Fatalf("could not isolate tag containing %q", marker)
	}
	return html[start : markerIndex+endOffset+1]
}

func TestAlertSubjectsByRuleType(t *testing.T) {
	tests := []struct {
		name                string
		data                AlertTemplateData
		triggered, resolved string
	}{
		{"service", AlertTemplateData{RuleType: "service_unhealthy", Severity: "critical", ServiceName: "Landing", ProjectName: "Test"}, "[Critical] Landing is unhealthy — Test", "[Resolved] Landing recovered — Test"},
		{"node", AlertTemplateData{RuleType: "node_offline", Severity: "critical", NodeName: "web-01", ProjectName: "Test"}, "[Critical] web-01 is offline — Test", "[Resolved] web-01 is back online — Test"},
		{"cpu", AlertTemplateData{RuleType: "cpu_above_threshold", Severity: "warning", MetricLabel: "CPU usage", ThresholdValue: floatPtr(80), NodeName: "web-01", ProjectName: "Test"}, "[Warning] CPU usage above 80% on web-01 — Test", "[Resolved] CPU usage back within threshold on web-01 — Test"},
		{"memory", AlertTemplateData{RuleType: "memory_above_threshold", Severity: "warning", MetricLabel: "Memory usage", ThresholdValue: floatPtr(90), NodeName: "web-01", ProjectName: "Test"}, "[Warning] Memory usage above 90% on web-01 — Test", "[Resolved] Memory usage back within threshold on web-01 — Test"},
		{"disk", AlertTemplateData{RuleType: "disk_above_threshold", Severity: "warning", MetricLabel: "Disk usage", ThresholdValue: floatPtr(85), NodeName: "web-01", ProjectName: "Test"}, "[Warning] Disk usage above 85% on web-01 — Test", "[Resolved] Disk usage back within threshold on web-01 — Test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := alertTriggeredEmailDefinition().subject(test.data); got != test.triggered {
				t.Errorf("triggered subject = %q, want %q", got, test.triggered)
			}
			if got := alertResolvedEmailDefinition().subject(test.data); got != test.resolved {
				t.Errorf("resolved subject = %q, want %q", got, test.resolved)
			}
		})
	}
}

func TestTriggeredHTTPHTMLAndTextContainOperationalContextAndCTAs(t *testing.T) {
	data := baseAlertData()
	renderer := NewRenderer()
	html, err := renderer.RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	text, err := renderer.RenderText("alert_triggered.txt", data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `data-title-icon="triggered"`) || !strings.Contains(html, "Landing is unhealthy") || !strings.Contains(text, "CRITICAL — Landing is unhealthy") {
		t.Fatal("triggered headline missing from HTML or plain text")
	}
	triggeredIconTag := startTagWithMarker(t, html, `data-title-icon="triggered"`)
	if !strings.Contains(triggeredIconTag, "color:#b42318") || !strings.Contains(html, ".triggered-semantic{color:#f17667!important}") {
		t.Fatal("triggered title icon is not using the active red state colour")
	}
	if !strings.Contains(html, `data-cta="primary"`) || !strings.Contains(html, `data-cta="primary" role="presentation" width="100%"`) || !strings.Contains(html, "background-color:#c94f40") {
		t.Fatal("triggered primary CTA is not a full-width red action")
	}
	primaryIndex := strings.Index(html, `data-cta="primary"`)
	secondaryIndex := strings.Index(html, `data-cta="secondary"`)
	if secondaryIndex <= primaryIndex || !strings.Contains(html, `data-cta="secondary" role="presentation" width="100%"`) || !strings.Contains(html, "Open Alerts") {
		t.Fatal("triggered Open Alerts CTA is not a separate full-width secondary action")
	}
	if !strings.Contains(html, "What this means") || !strings.Contains(html, "Agrafa is unable to successfully complete the configured HTTP health check for this service.") {
		t.Fatal("service explanation block missing or incorrect")
	}
	for _, value := range []string{"Status code", "503", "Response time", "412 ms", "Failure reason", "Service Unavailable", "Service unhealthy", data.ResourceURL, data.AlertsURL} {
		if !strings.Contains(html, value) {
			t.Errorf("HTML missing %q", value)
		}
		if !strings.Contains(text, value) {
			t.Errorf("text missing %q", value)
		}
	}
	if strings.Contains(html, "service_unhealthy") || strings.Contains(text, "service_unhealthy") {
		t.Fatal("raw rule enum leaked into rendered email")
	}
}

func TestTriggeredHTMLRendersCTAsIndependently(t *testing.T) {
	tests := []struct {
		name                     string
		resourceURL, alertsURL   string
		wantResource, wantAlerts bool
	}{
		{"both", "https://app.agrafa.test/services/13?project_id=7", "https://app.agrafa.test/alerts?project_id=7", true, true},
		{"resource only", "https://app.agrafa.test/services/13?project_id=7", "", true, false},
		{"alerts only", "", "https://app.agrafa.test/alerts?project_id=7", false, true},
		{"neither", "", "", false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := baseAlertData()
			data.ResourceURL = test.resourceURL
			data.AlertsURL = test.alertsURL
			output, err := NewRenderer().RenderHTML("alert_triggered.html", data)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(output, `data-cta="primary"`); got != test.wantResource {
				t.Errorf("primary CTA rendered = %v, want %v", got, test.wantResource)
			}
			if got := strings.Contains(output, `data-cta="secondary"`); got != test.wantAlerts {
				t.Errorf("Open Alerts CTA rendered = %v, want %v", got, test.wantAlerts)
			}
			if test.wantResource && (!strings.Contains(output, test.resourceURL) || !strings.Contains(output, "View service in Agrafa")) {
				t.Error("resource CTA lost its URL or copy")
			}
			if test.wantAlerts && (!strings.Contains(output, test.alertsURL) || !strings.Contains(output, "Open Alerts")) {
				t.Error("Open Alerts CTA lost its URL or copy")
			}
		})
	}
}

func TestTriggeredTCPAndEmptyOptionalsOmitHTTPRows(t *testing.T) {
	data := baseAlertData()
	data.ServiceName = "Database"
	data.AlertTitle = "⚠ Database is unhealthy"
	data.AlertMessage = "TCP connection to db.internal:5432 failed."
	data.ServiceCheckType = "TCP"
	data.ServiceTarget = "db.internal:5432"
	data.StatusCode = nil
	data.ResponseTimeMs = nil
	data.FailureReason = "Connection refused"
	html, err := NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "Status code") || strings.Contains(html, "Response time") {
		t.Fatalf("TCP HTML rendered HTTP-only rows: %s", html)
	}
	if !strings.Contains(html, "Connection refused") || !strings.Contains(html, "db.internal:5432") {
		t.Fatal("TCP HTML omitted useful failure context")
	}
}

func TestAlertHTMLPreventsDefaultBlueLinkStyling(t *testing.T) {
	data := baseAlertData()
	data.ServiceTarget = "http://localhost:3000"
	data.FailureReason = `Get "http://localhost:3000": dial tcp: connection refused`

	triggered, err := NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := NewRenderer().RenderHTML("alert_resolved.html", data)
	if err != nil {
		t.Fatal(err)
	}

	autoDetectionRule := "a[x-apple-data-detectors],u + #body a,#MessageViewBody a{color:inherit!important;text-decoration:inherit!important;text-decoration-color:inherit!important;}"
	for templateName, output := range map[string]string{"triggered": triggered, "resolved": resolved} {
		if !strings.Contains(output, autoDetectionRule) || !strings.Contains(output, `body id="body"`) {
			t.Errorf("%s template is missing email-client auto-detection protection", templateName)
		}
		footerTag := startTagWithMarker(t, output, `class="agrafa-footer-link"`)
		if !strings.Contains(footerTag, "color:#6b6f75!important") || !strings.Contains(footerTag, "text-decoration-color:#a5a7aa!important") || !strings.Contains(output, ".agrafa-footer-link{color:#8d9198!important;text-decoration-color:#555960!important}") {
			t.Errorf("%s footer link lacks explicit restrained styling: %s", templateName, footerTag)
		}
		primaryTag := startTagWithMarker(t, output, `class="agrafa-primary-link"`)
		if !strings.Contains(primaryTag, "color:#ffffff!important") || !strings.Contains(primaryTag, "text-decoration:none!important") {
			t.Errorf("%s primary CTA did not preserve its explicit styling: %s", templateName, primaryTag)
		}
	}

	targetTag := startTagWithMarker(t, triggered, `data-link-scope="target"`)
	if !strings.Contains(targetTag, "color:#343638") || !strings.Contains(targetTag, "text-decoration:underline") || !strings.Contains(targetTag, "text-decoration-color:#a1a3a6") || !strings.Contains(triggered, ".target-value{color:#d9dade!important;text-decoration-color:#555960!important}") {
		t.Fatalf("target URL container lacks explicit neutral link styling: %s", targetTag)
	}
	failureTag := startTagWithMarker(t, triggered, `data-link-scope="failure"`)
	if !strings.Contains(failureTag, "color:#b42318") || !strings.Contains(failureTag, "text-decoration-color:#c78a84") || !strings.Contains(triggered, ".failure-value{color:#f17667!important;text-decoration-color:#6f3731!important}") {
		t.Fatalf("failure URL container does not protect its semantic error colour: %s", failureTag)
	}
	secondaryTag := startTagWithMarker(t, triggered, `class="agrafa-secondary-link"`)
	if !strings.Contains(secondaryTag, "color:#a93228!important") || !strings.Contains(secondaryTag, "text-decoration:none!important") || !strings.Contains(triggered, "border:1px solid #b9382c") || !strings.Contains(triggered, ".agrafa-secondary-link{color:#ef7768!important}") {
		t.Fatalf("Open Alerts CTA did not preserve its red outlined styling: %s", secondaryTag)
	}
	if !strings.Contains(triggered, "background-color:#b9382c") || !strings.Contains(triggered, ".triggered-primary-cell{background-color:#c94f40!important") || !strings.Contains(resolved, "background-color:#2f8a4b") || !strings.Contains(resolved, ".resolved-primary-cell{background-color:#3f9859!important") {
		t.Fatal("triggered/resolved CTA background colours changed")
	}
}

func TestAlertHTMLDefinesIntentionalLightAndDarkThemes(t *testing.T) {
	data := baseAlertData()
	triggered, err := NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := NewRenderer().RenderHTML("alert_resolved.html", data)
	if err != nil {
		t.Fatal(err)
	}

	for templateName, output := range map[string]string{"triggered": triggered, "resolved": resolved} {
		for _, declaration := range []string{
			`meta name="color-scheme" content="light dark"`,
			`meta name="supported-color-schemes" content="light dark"`,
			":root{color-scheme:light dark;supported-color-schemes:light dark;}",
			"@media (prefers-color-scheme:dark)",
			`class="email-body"`,
			"background-color:#f5f4f1",
			`class="email-card"`,
			"background-color:#ffffff",
			".email-body,.email-page{background-color:#111214!important;color:#f2f3f5!important}",
			".email-card{background-color:#1a1c1f!important;border-color:#2a2d31!important}",
			".primary-text",
			".metadata-strip{border-color:#2d3034!important}",
		} {
			if !strings.Contains(output, declaration) {
				t.Errorf("%s template missing theme declaration %q", templateName, declaration)
			}
		}
	}

	for _, treatment := range []string{
		"background-color:#fff4f2",
		"border-left:3px solid #c43d2f",
		".explanation-panel{background-color:#211b1b!important",
		".triggered-pill{background-color:#2b2020!important",
		".triggered-primary-cell{background-color:#c94f40!important",
	} {
		if !strings.Contains(triggered, treatment) {
			t.Errorf("triggered template missing light/dark treatment %q", treatment)
		}
	}
	for _, treatment := range []string{
		"background-color:#f0f8f2",
		"border-left:3px solid #2f8a4b",
		".explanation-panel{background-color:#19221c!important",
		"background-color:#fff8e8",
		"border:1px solid #e4c981",
		".downtime-panel{background-color:#1d1d1b!important;border-color:#4b4330!important}",
		".resolved-primary-cell{background-color:#3f9859!important",
	} {
		if !strings.Contains(resolved, treatment) {
			t.Errorf("resolved template missing light/dark treatment %q", treatment)
		}
	}
}

func TestNotificationRecipientTestHTMLUsesAlertLightAndDarkThemes(t *testing.T) {
	output, err := NewRenderer().RenderHTML("notification_recipient_test.html", NotificationRecipientTestTemplateData{
		ProjectName: "Test", Recipient: "ops@example.com", SentAt: time.Date(2026, 8, 29, 8, 9, 57, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range []string{
		`meta name="color-scheme" content="light dark"`,
		`meta name="supported-color-schemes" content="light dark"`,
		"background-color:#f5f4f1",
		"background-color:#ffffff",
		"@media (prefers-color-scheme:dark)",
		".email-body,.email-page{background-color:#111214!important",
		".email-card{background-color:#1a1c1f!important",
	} {
		if !strings.Contains(output, declaration) {
			t.Errorf("notification test template missing theme declaration %q", declaration)
		}
	}
}

func TestMetricHTMLAndTextRenderCurrentValueAndThreshold(t *testing.T) {
	data := baseAlertData()
	data.RuleType = "cpu_above_threshold"
	data.RuleLabel = "CPU above threshold"
	data.ResourceType = "node"
	data.NodeName = "web-01"
	data.MetricLabel = "CPU usage"
	data.MetricValue = floatPtr(91)
	data.ThresholdValue = floatPtr(80)
	data.AlertTitle = "⚠ CPU usage is high on web-01"
	data.AlertMessage = "CPU usage reached 91%, above the configured threshold of 80%."
	for _, templateName := range []string{"alert_triggered.html", "alert_triggered.txt"} {
		var output string
		var err error
		if strings.HasSuffix(templateName, ".html") {
			output, err = NewRenderer().RenderHTML(templateName, data)
		} else {
			output, err = NewRenderer().RenderText(templateName, data)
		}
		if err != nil || !strings.Contains(output, "Current value") || !strings.Contains(output, "91%") || !strings.Contains(output, "Threshold") || !strings.Contains(output, "80%") {
			t.Fatalf("metric template %s missing values (err=%v): %s", templateName, err, output)
		}
	}
}

func TestResolvedHTMLUsesAmberDowntimeAndOnlyPrimaryCTA(t *testing.T) {
	data := baseAlertData()
	data.Status = "resolved"
	data.AlertTitle = "✓ Landing has recovered"
	data.AlertMessage = "HTTP health checks are passing again. The service is responding normally."
	html, err := NewRenderer().RenderHTML("alert_resolved.html", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"RESOLVED", `data-title-icon="resolved"`, "color:#217a3c", ".resolved-semantic{color:#72cf8b!important}", "Landing has recovered", "Recovery details", "20m 31s", `data-downtime-callout="amber"`, `data-duration-color="amber"`, "◷", "color:#a86600", ".downtime-semantic{color:#e7b24c!important}", data.ResourceURL, "View service in Agrafa", "What happened", "The configured health check is passing again and the service is responding normally."} {
		if !strings.Contains(html, value) {
			t.Errorf("resolved HTML missing %q", value)
		}
	}
	if !strings.Contains(html, `data-cta="primary" role="presentation" width="100%"`) || !strings.Contains(html, "background-color:#3f9859") {
		t.Fatal("resolved primary CTA is not a full-width green action")
	}
	if strings.Contains(html, "Open Alerts") || strings.Contains(html, data.AlertsURL) {
		t.Fatal("resolved email unexpectedly contains secondary alerts CTA")
	}
}

func TestAlertExplanationBlocksUseRuleSpecificCopy(t *testing.T) {
	tests := []struct {
		name, templateName, ruleType, metricLabel, want string
	}{
		{"triggered node", "alert_triggered.html", "node_offline", "", "Agrafa stopped receiving heartbeats from this node and has marked it offline."},
		{"triggered metric", "alert_triggered.html", "cpu_above_threshold", "CPU usage", "The latest CPU usage exceeded the configured alert threshold."},
		{"resolved node", "alert_resolved.html", "node_offline", "", "The node resumed sending heartbeats and is communicating normally again."},
		{"resolved metric", "alert_resolved.html", "cpu_above_threshold", "CPU usage", "CPU usage returned within the configured alert threshold."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := baseAlertData()
			data.RuleType = test.ruleType
			data.MetricLabel = test.metricLabel
			data.ResourceType = "node"
			output, err := NewRenderer().RenderHTML(test.templateName, data)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, test.want) {
				t.Fatalf("explanation missing %q", test.want)
			}
		})
	}
}

func TestTriggeredNodeOmitsDuplicateOrEmptyIdentifier(t *testing.T) {
	data := baseAlertData()
	data.RuleType = "node_offline"
	data.RuleLabel = "Node offline"
	data.ResourceType = "node"
	data.NodeName = "web-01"
	data.NodeIdentifier = "web-01"
	data.AlertTitle = "⚠ web-01 is offline"
	data.AlertMessage = "Agrafa stopped receiving heartbeats from this node."

	html, err := NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, ">Identifier<") {
		t.Fatal("duplicate node identifier row was rendered")
	}

	data.NodeIdentifier = ""
	html, err = NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, ">Identifier<") {
		t.Fatal("empty node identifier row was rendered")
	}

	data.NodeIdentifier = "web-01.internal"
	html, err = NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, ">Identifier<") || !strings.Contains(html, "web-01.internal") {
		t.Fatal("distinct node identifier row was omitted")
	}
}

func TestAlertHTMLEscapesUntrustedPresentationValues(t *testing.T) {
	data := baseAlertData()
	data.AlertTitle = `⚠ <script>alert("x")</script> is unhealthy`
	data.AlertMessage = `Check to https://example.test/?a=1&b=2 failed <hard>.`
	data.ServiceTarget = `https://example.test/?a=1&b=<bad>`
	data.FailureReason = `<img src=x onerror=alert(1)>`
	html, err := NewRenderer().RenderHTML("alert_triggered.html", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<script>") || strings.Contains(html, "<img src=x") || strings.Contains(html, "b=<bad>") {
		t.Fatalf("unescaped content in alert HTML: %s", html)
	}
	for _, escaped := range []string{"&lt;script&gt;", "&lt;img", "&amp;b="} {
		if !strings.Contains(html, escaped) {
			t.Errorf("expected escaped value %q", escaped)
		}
	}
}

func TestResolvedPlainTextHasTruthfulRecoveryParity(t *testing.T) {
	data := baseAlertData()
	data.Status = "resolved"
	data.AlertTitle = "✓ Landing has recovered"
	data.AlertMessage = "HTTP health checks are passing again. The service is responding normally."
	text, err := NewRenderer().RenderText("alert_resolved.txt", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"RESOLVED", "Landing has recovered", "Recovered at:", "Downtime: 20m 31s", "Rule: Service unhealthy", data.ResourceURL, data.NotificationsURL} {
		if !strings.Contains(text, value) {
			t.Errorf("resolved text missing %q", value)
		}
	}
	if strings.Contains(text, "is unhealthy") {
		t.Fatal("resolved plain text repeats triggered failure copy")
	}
}
