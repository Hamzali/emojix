let eventSource
window.onload = function() {
	console.log('currentPath', window.location.pathname)

	const ssePath = `${window.location.pathname}/sse`


	eventSource = new EventSource(ssePath)

	eventSource.addEventListener("ping", () => {
		console.log('this ping is coming from server!')
	})
	eventSource.addEventListener("msg", (event) => {
		console.log('msg data', event.data)
	})
}

window.onbeforeunload = function() {
	if (eventSource == null) {
		return
	}
	eventSource.close()

}
