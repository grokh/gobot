// The build designer's script. As a slot's box is typed in, it offers the
// names of the items that fit there. Each change is posted as the page's own
// buttons post one, and the page that comes back takes this one's place, so
// the totals change without the page reloading: the address bar keeps the
// build's link, and the back button steps back through the changes. Without
// it, the page works by its buttons alone. grokh.com allows no inline script
// or style, so all of it is here, and style.css places the list of names.
(function () {
	'use strict';
	if (!window.fetch || !window.DOMParser || !window.URLSearchParams ||
		!window.FormData || !window.history.pushState ||
		!Element.prototype.closest) {
		return;
	}

	var sent = 0;      // the number of the latest page asked for
	var lastBody = ''; // what was posted last, while it's on its way
	var bypass = false; // let the next submit go the usual way
	var clicked = null; // the button clicked, where submit doesn't say
	var timer = null;  // the wait before asking for names
	var asking = null; // the AbortController of the latest ask for names
	var list = null;   // the list of names on show
	var box = null;    // the box it's for
	var on = -1;       // the name picked out with the arrow keys

	function form() {
		return document.getElementById('build');
	}

	// show puts the page in html in place of this one: the part that holds
	// the form, so this script and its handlers stay. The box with the focus
	// keeps it, and what's been typed in it since the change was sent, unless
	// it's the box whose change was sent, which shows the item found.
	function show(html, url, from, push) {
		var doc = new DOMParser().parseFromString(html, 'text/html');
		var fresh = doc.getElementById('designer');
		var old = document.getElementById('designer');
		if (!fresh || !old) {
			window.location.href = url;
			return;
		}
		closeList();
		var active = document.activeElement;
		var keep = null;
		if (active && active.id && old.contains(active)) {
			keep = {id: active.id, value: active.value, sent: active === from};
			try {
				keep.start = active.selectionStart;
				keep.end = active.selectionEnd;
			} catch (e) {
				keep.start = null;
			}
		}
		old.parentNode.replaceChild(document.importNode(fresh, true), old);
		if (push && url !== window.location.href) {
			window.history.pushState(null, '', url);
		}
		if (!keep) {
			return;
		}
		var el = document.getElementById(keep.id);
		if (!el) {
			return;
		}
		if (!keep.sent && el.matches('input[type=text], textarea')) {
			el.value = keep.value;
		}
		el.focus();
		if (!keep.sent && typeof keep.start === 'number' && el.setSelectionRange) {
			try {
				el.setSelectionRange(keep.start, keep.end);
			} catch (e) {
				// not a box with a caret
			}
		}
	}

	// send posts the form, with the button it was sent by, and shows the page
	// the post is redirected to. If that fails, the form is sent the usual
	// way, and the browser shows the page.
	function send(button, from) {
		var f = form();
		var data = new FormData(f);
		if (button && button.name) {
			data.append(button.name, button.value);
		}
		var body = new URLSearchParams(data).toString();
		if (body === lastBody) {
			return; // a change and the Enter that made it: one post will do
		}
		lastBody = body;
		var n = ++sent;
		closeList();
		fetch(f.action, {
			method: 'POST',
			body: body,
			headers: {'Content-Type': 'application/x-www-form-urlencoded'}
		}).then(function (res) {
			if (!res.ok) {
				throw new Error('status ' + res.status);
			}
			return res.text().then(function (html) {
				if (n === sent) {
					lastBody = '';
					show(html, res.url, from, true);
				}
			});
		}).catch(function () {
			if (n === sent) {
				lastBody = '';
				bypass = true;
				if (button && f.requestSubmit) {
					f.requestSubmit(button);
				} else {
					f.submit();
				}
			}
		});
	}

	// load shows the page at the address the back or forward button took the
	// browser to.
	function load() {
		var n = ++sent;
		lastBody = '';
		fetch(window.location.href).then(function (res) {
			if (!res.ok) {
				throw new Error('status ' + res.status);
			}
			return res.text();
		}).then(function (html) {
			if (n === sent) {
				show(html, window.location.href, null, false);
			}
		}).catch(function () {
			window.location.reload();
		});
	}

	function closeList() {
		if (list && list.parentNode) {
			list.parentNode.removeChild(list);
		}
		if (box) {
			box.setAttribute('aria-expanded', 'false');
			box.removeAttribute('aria-activedescendant');
		}
		list = null;
		box = null;
		on = -1;
	}

	// showList shows the names found for the box el under it.
	function showList(el, names) {
		closeList();
		if (!names.length) {
			return;
		}
		list = document.createElement('ul');
		list.className = 'suggest';
		list.id = 'suggest';
		list.setAttribute('role', 'listbox');
		names.forEach(function (item, i) {
			var li = document.createElement('li');
			li.id = 'suggest-' + i;
			li.setAttribute('role', 'option');
			li.setAttribute('data-name', item.name);
			li.appendChild(document.createTextNode(item.name));
			if (item.stats) {
				var stats = document.createElement('span');
				stats.textContent = ' ' + item.stats;
				li.appendChild(stats);
			}
			list.appendChild(li);
		});
		el.parentNode.appendChild(list);
		box = el;
		el.setAttribute('aria-expanded', 'true');
		el.setAttribute('aria-controls', 'suggest');
	}

	// ask asks for the names to offer for what's typed in the box el.
	function ask(el) {
		var q = el.value.trim();
		if (asking) {
			asking.abort();
			asking = null;
		}
		if (q.length < 2) {
			closeList();
			return;
		}
		var f = form();
		var params = new URLSearchParams();
		params.set('slot', el.getAttribute('data-slot'));
		params.set('q', q);
		['class', 'race', 'align', 'sex'].forEach(function (k) {
			var field = f.elements[k];
			if (field && field.value) {
				params.set(k, field.value);
			}
		});
		var opts = {};
		if (window.AbortController) {
			asking = new AbortController();
			opts.signal = asking.signal;
		}
		fetch('build/names?' + params.toString(), opts).then(function (res) {
			return res.ok ? res.json() : [];
		}).then(function (names) {
			if (document.activeElement === el) {
				showList(el, names);
			}
		}).catch(function () {
			// a newer ask took its place, or the names can't be had
		});
	}

	// pick puts a name from the list in its box, and sends the change.
	function pick(li) {
		var el = box;
		el.value = li.getAttribute('data-name');
		closeList();
		send(null, el);
	}

	// highlight picks out the nth name in the list.
	function highlight(n) {
		var items = list.children;
		for (var i = 0; i < items.length; i++) {
			items[i].className = i === n ? 'on' : '';
		}
		on = n;
		box.setAttribute('aria-activedescendant', items[n].id);
		if (items[n].scrollIntoView) {
			items[n].scrollIntoView({block: 'nearest'});
		}
	}

	document.addEventListener('input', function (e) {
		if (!e.target.matches('#build input.item')) {
			return;
		}
		var el = e.target;
		clearTimeout(timer);
		timer = setTimeout(function () {
			ask(el);
		}, 150);
	});

	document.addEventListener('keydown', function (e) {
		if (!list || e.target !== box) {
			return;
		}
		var count = list.children.length;
		switch (e.key) {
		case 'ArrowDown':
			e.preventDefault();
			highlight(on < count - 1 ? on + 1 : 0);
			break;
		case 'ArrowUp':
			e.preventDefault();
			highlight(on > 0 ? on - 1 : count - 1);
			break;
		case 'Enter':
			if (on >= 0) {
				e.preventDefault();
				pick(list.children[on]);
			}
			break;
		case 'Escape':
			closeList();
			break;
		}
	});

	// a name is picked with the mouse before the box loses the focus
	document.addEventListener('mousedown', function (e) {
		var li = e.target.closest('#suggest li');
		if (li) {
			e.preventDefault();
			pick(li);
		}
	});

	document.addEventListener('focusout', function (e) {
		if (e.target === box) {
			closeList();
		}
	});

	document.addEventListener('click', function (e) {
		var b = e.target.closest('button, input[type=submit]');
		clicked = b && b.form === form() ? b : null;
	});

	// a box or list changed: send it, but not the pasted list or what a fill
	// is asked for, which wait for their buttons
	document.addEventListener('change', function (e) {
		var el = e.target;
		if (el.form !== form() || el.name === 'paste' || el.name === 'pasteTo' ||
			el.closest('.fill')) {
			return;
		}
		send(null, el);
	});

	document.addEventListener('submit', function (e) {
		if (bypass || e.target !== form()) {
			bypass = false;
			return;
		}
		e.preventDefault();
		var button = e.submitter || clicked;
		clicked = null;
		// Enter in a box sends that box's change
		var from = document.activeElement;
		send(button, from && from.form === form() ? from : null);
	});

	window.addEventListener('popstate', load);
})();
