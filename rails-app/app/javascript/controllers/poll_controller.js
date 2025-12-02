import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  static values = { url: String }

  connect() {
    this.startPolling()
  }

  disconnect() {
    this.stopPolling()
  }

  startPolling() {
    this.interval = setInterval(() => {
      this.fetchStatus()
    }, 2000)
  }

  stopPolling() {
    if (this.interval) clearInterval(this.interval)
  }

  fetchStatus() {
    fetch(this.urlValue, {
      headers: { "Accept": "text/vnd.turbo-stream.html" }
    })
    .then(response => response.text())
    .then(html => {
        // Always render the update
        Turbo.renderStreamMessage(html)
        
        // Stop polling if complete
        if (html.includes("complete-flag")) {
            this.stopPolling()
        }
    })
  }
}
