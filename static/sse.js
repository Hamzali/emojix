console.log('currentPath', window.location.pathname)

const ssePath = `${window.location.pathname}/sse`


const eventSource = new EventSource(ssePath)

eventSource.addEventListener("ping", () => {
	console.log('this ping is coming from server!')
})
