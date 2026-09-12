package main

import (
	"fmt"
	"html"
	"net/http"
	"strings"
)

const enabledLandingPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>AmiGuard Sample Submission</title>
</head>
<body>
<main>
<h1>AmiGuard Sample Submission</h1>
<p>Submit suspected Amiga, Atari ST, or classic 68k Macintosh malware for defensive research.</p>

<h2>Before you submit</h2>
<ul>
<li>Only submit files or disk images that you are authorized to provide for malware research.</li>
<li>Do not submit personal documents, credentials, private communications, or unrelated data.</li>
<li>Maximum sample size: 16 MiB.</li>
<li>The original filename is not retained. The service assigns a random submission ID and records the platform, SHA-256 digest, size, receipt time, and consent state.</li>
<li>Submissions are quarantined and are not made available through a public download endpoint.</li>
<li>Unclassified submissions are normally retained for up to 90 days. Material retained as research evidence may be kept longer when necessary for malware research and signature development.</li>
</ul>

<form method="post" action="/api/v1/submissions" enctype="multipart/form-data">
<p><label>Platform<br>
<select name="platform" required>
<option value="amiga">Amiga</option>
<option value="atari-st">Atari ST / STE</option>
<option value="mac68k">Classic Macintosh (68k)</option>
</select>
</label></p>
<p><label>Sample<br><input type="file" name="sample" required></label></p>
<p><label><input type="checkbox" name="consent" value="true" required> I confirm that I am authorized to submit this material for defensive malware research and consent to its quarantine, analysis, and retention under the policy above.</label></p>
<p><button type="submit">Submit sample</button></p>
</form>

<p>The receipt returned after a successful submission contains the submission ID, platform, SHA-256 digest, size, and receipt time. Keep the ID if you need to refer to the submission later.</p>
<p>AmiGuard does not execute, extract, or publicly serve samples as part of the intake process.</p>
</main>
</body>
</html>`

const disabledLandingPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>AmiGuard Sample Submission</title>
</head>
<body>
<main>
<h1>AmiGuard Sample Submission</h1>
<p>The secure submission channel is currently closed.</p>
<p>No samples are being accepted at this time.</p>
</main>
</body>
</html>`

func landingPage(uploadEnabled bool) string {
	if uploadEnabled {
		return enabledLandingPage
	}
	return disabledLandingPage
}

func wantsHTMLReceipt(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "application/xhtml+xml")
}

func receiptPage(receipt submissionReceipt) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>AmiGuard Submission Receipt</title>
</head>
<body>
<main>
<h1>Submission received</h1>
<p>Your sample was accepted and stored in quarantine.</p>
<h2>Receipt</h2>
<dl>
<dt>Submission ID</dt><dd><code>%s</code></dd>
<dt>Platform</dt><dd><code>%s</code></dd>
<dt>SHA-256</dt><dd><code>%s</code></dd>
<dt>Size</dt><dd>%d bytes</dd>
<dt>Received</dt><dd><time datetime="%s">%s</time></dd>
</dl>
<p>Keep the submission ID and SHA-256 digest if you need to refer to this sample later.</p>
<p>The intake service does not provide a public download or retrieval endpoint.</p>
<p><a href="/">Submit another sample</a></p>
</main>
</body>
</html>`,
		html.EscapeString(receipt.ID),
		html.EscapeString(receipt.Platform),
		html.EscapeString(receipt.SHA256),
		receipt.Size,
		html.EscapeString(receipt.ReceivedAt),
		html.EscapeString(receipt.ReceivedAt),
	)
}
