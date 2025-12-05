let eventSource
window.onload = function() {
	console.log('currentPath', window.location.pathname)

	const ssePath = `${window.location.pathname}/sse`


	eventSource = new EventSource(ssePath)

	eventSource.addEventListener("open", (event) => {
		console.log('sse connected', event)
	})

	eventSource.addEventListener("ping", () => {
		console.log('this ping is coming from server!')
	})

	eventSource.addEventListener("msg", (event) => {
		const msgData = event.data
		const [nickname, message] = msgData.split(",", 2)
		console.log('msg data', nickname, message)

		const messagesContainer = document.getElementById("messages")

		const newMessageNode = document.createElement("p")
		newMessageNode.textContent = `${nickname} ${message}`

		messagesContainer.prepend(newMessageNode)
	})
}

window.onbeforeunload = function() {
	if (eventSource == null) {
		return
	}
	eventSource.close()

}
