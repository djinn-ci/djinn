package http

var (
	badgeUnknown = `<svg xmlns="http://www.w3.org/2000/svg" width="115" height="20">
	<title>Djinn CI: unknown</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="60" height="20" fill="#6a7393"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="83" y="14">unknown</text>
	</g>
</svg>`

	badgeQueued = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: queued</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#272b39"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">queued</text>
	</g>
</svg>`
	badgeRunning = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: running</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#61a0ea"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">running</text>
	</g>
</svg>`
	badgePassed = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: passed</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#269326"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">passed</text>
	</g>
</svg>`
	badgePassedWithFailures = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: passed</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#ff7400"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">passed</text>
	</g>
</svg>`
	badgeFailed = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: failed</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#c64242"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">failed</text>
	</g>
</svg>`
	badgeKilled = `<svg xmlns="http://www.w3.org/2000/svg" width="105" height="20">
	<title>Djinn CI: killed</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="54" height="20" fill="#c64242"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="78" y="14">killed</text>
	</g>
</svg>`
	badgeTimedOut = `<svg xmlns="http://www.w3.org/2000/svg" width="115" height="20">
	<title>Djinn CI: timed out</title>
	<rect width="100" height="20" fill="#383e51"/>
	<rect x="53" width="63" height="20" fill="#6a7393"/>
	<rect width="100" height="20" fill="url(#a)"/>
	<g fill="#fff" text-anchor="middle" font-family="sans-serif" font-size="11">
		<text x="25" y="14">Djinn CI</text>
		<text x="83" y="14">timed out</text>
	</g>
</svg>`
)
