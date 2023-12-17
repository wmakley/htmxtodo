function getCsrfToken() {
	return getCookie("htmxtodo_csrf");
}

function getCookie(c_name) {
	let c_value = " " + document.cookie;
	let c_start = c_value.indexOf(" " + c_name + "=");
	if (c_start === -1) {
		c_value = null;
	} else {
		c_start = c_value.indexOf("=", c_start) + 1;
		let c_end = c_value.indexOf(";", c_start);
		if (c_end === -1) {
			c_end = c_value.length;
		}
		c_value = decodeURIComponent(c_value.substring(c_start, c_end));
	}
	return c_value;
}

document.addEventListener('DOMContentLoaded', function () {
	// add X-CSRF-Token to all non-GET requests:
	document.body.addEventListener('htmx:configRequest', function (event) {
		if (event.detail.verb !== "get") {
			event.detail.headers['X-CSRF-Token'] = getCsrfToken();
		}
	});

	// treat 204 "no content" same as 200 "success":
	document.body.addEventListener('htmx:beforeSwap', function (event) {
		const status = event.detail.xhr.status;
		if (status === 204 || status === 422) {
			event.detail.shouldSwap = true;
		}
	});
});
