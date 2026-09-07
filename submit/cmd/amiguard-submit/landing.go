package main

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
<p>Submit suspected Amiga malware for defensive research and possible future AmiGuard detection.</p>

<h2>Before you submit</h2>
<ul>
<li>Only submit files or disk images that you are authorized to provide for malware research.</li>
<li>Do not submit personal documents, credentials, private communications, or unrelated data.</li>
<li>Maximum sample size: 16 MiB.</li>
<li>The original filename is not retained. The service assigns a random submission ID and records a SHA-256 digest, size, receipt time, and consent state.</li>
<li>Submissions are quarantined and are not made available through a public download endpoint.</li>
<li>Unclassified submissions are normally retained for up to 90 days. Material retained as research evidence may be kept longer when necessary for AmiGuard development.</li>
</ul>

<form method="post" action="/api/v1/submissions" enctype="multipart/form-data">
<p><label>Sample<br><input type="file" name="sample" required></label></p>
<p><label><input type="checkbox" name="consent" value="true" required> I confirm that I am authorized to submit this material for defensive malware research and consent to its quarantine, analysis, and retention under the policy above.</label></p>
<p><button type="submit">Submit sample</button></p>
</form>

<p>The receipt returned after a successful submission contains the submission ID, SHA-256 digest, size, and receipt time. Keep the ID if you need to refer to the submission later.</p>
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
