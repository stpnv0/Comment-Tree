'use strict';

var API = '/comments';
var PAGE_SIZE = 20;

var treePage = 1;
var searchPage = 1;

// --- API helper ---
// FIX: handle 204 No Content (empty body from DELETE)
async function api(url, opts = {}) {
    var hasBody = opts.body !== undefined;
    var resp = await fetch(url, {
        ...(hasBody ? { headers: { 'Content-Type': 'application/json' } } : {}),
        ...opts
    });

    if (!resp.ok) {
        var data = await resp.json().catch(function() { return {}; });
        throw new Error(data.error || 'HTTP ' + resp.status);
    }

    // 204 No Content — no body to parse
    if (resp.status === 204 || resp.headers.get('content-length') === '0') {
        return null;
    }

    return resp.json();
}

function showError(msg) {
    var el = document.getElementById('errorMsg');
    el.textContent = msg;
    el.style.display = 'block';
    setTimeout(function() { el.style.display = 'none'; }, 5000);
}

// --- Tree ---

async function loadTree(page) {
    if (page === undefined) page = 1;
    treePage = page;
    try {
        var limit = PAGE_SIZE;
        var offset = (page - 1) * limit;

        var data = await api(
            API + '?limit=' + limit + '&offset=' + offset + '&sort_by=path&sort_order=asc'
        );

        renderTree(data.comments || []);
        renderPagination('treePagination', data.total || 0, page, limit, 'loadTree');
    } catch (e) {
        showError(e.message);
    }
}

function buildTree(comments) {
    var idMap = {};
    var roots = [];

    comments.forEach(function(c) {
        c._children = [];
        idMap[c.id] = c;
    });

    comments.forEach(function(c) {
        if (c.parent_id && idMap[c.parent_id]) {
            idMap[c.parent_id]._children.push(c);
        } else {
            roots.push(c);
        }
    });

    return roots;
}

function renderTree(comments) {
    var roots = buildTree(comments);
    var container = document.getElementById('commentTree');
    container.innerHTML = '';

    if (roots.length === 0) {
        container.innerHTML = '<div class="empty-state">No comments yet. Be the first!</div>';
        return;
    }

    roots.forEach(function(c) {
        container.appendChild(renderComment(c));
    });
}

function renderComment(c) {
    var div = document.createElement('div');
    var comment = document.createElement('div');
    comment.className = 'comment';

    var date = new Date(c.created_at).toLocaleString();

    comment.innerHTML =
        '<div class="comment-header">' +
        '<span class="comment-author">' + esc(c.author) + '</span>' +
        '<span class="comment-date">' + esc(date) + '</span>' +
        '</div>' +
        '<div class="comment-body">' + esc(c.body) + '</div>' +
        '<div class="comment-actions">' +
        '<button class="btn btn-outline btn-sm" onclick="toggleReply(' + c.id + ')">Reply</button>' +
        '<button class="btn btn-danger btn-sm" onclick="deleteComment(' + c.id + ')">Delete</button>' +
        '</div>' +
        '<div id="reply-form-' + c.id + '" class="reply-form" style="display:none">' +
        '<div class="form-group">' +
        '<input type="text" id="reply-author-' + c.id + '" placeholder="Your name" maxlength="255">' +
        '</div>' +
        '<div class="form-group">' +
        '<textarea id="reply-body-' + c.id + '" placeholder="Write a reply..." maxlength="10000"></textarea>' +
        '</div>' +
        '<button class="btn btn-primary btn-sm" onclick="submitReply(' + c.id + ')">Send</button>' +
        '</div>';

    div.appendChild(comment);

    if (c._children && c._children.length > 0) {
        var children = document.createElement('div');
        children.className = 'children';
        c._children.forEach(function(ch) {
            children.appendChild(renderComment(ch));
        });
        div.appendChild(children);
    }

    return div;
}

function toggleReply(id) {
    var form = document.getElementById('reply-form-' + id);
    form.style.display = form.style.display === 'none' ? 'block' : 'none';
}

async function submitReply(parentId) {
    var author = document.getElementById('reply-author-' + parentId).value.trim();
    var body = document.getElementById('reply-body-' + parentId).value.trim();

    if (!author || !body) {
        showError('Name and comment text are required');
        return;
    }

    try {
        await api(API, {
            method: 'POST',
            body: JSON.stringify({ parent_id: parentId, author: author, body: body })
        });
        loadTree(treePage);
    } catch (e) {
        showError(e.message);
    }
}

async function createRootComment() {
    var author = document.getElementById('rootAuthor').value.trim();
    var body = document.getElementById('rootBody').value.trim();

    if (!author || !body) {
        showError('Name and comment text are required');
        return;
    }

    try {
        await api(API, {
            method: 'POST',
            body: JSON.stringify({ author: author, body: body })
        });
        document.getElementById('rootAuthor').value = '';
        document.getElementById('rootBody').value = '';
        loadTree(1);
    } catch (e) {
        showError(e.message);
    }
}

// FIX: after successful delete (204), reload tree immediately
async function deleteComment(id) {
    if (!confirm('Delete this comment and all replies?')) return;

    try {
        await api(API + '/' + id, { method: 'DELETE' });
        // api() returns null for 204 — no error, just reload
        loadTree(treePage);
    } catch (e) {
        showError(e.message);
    }
}

// --- Search ---

async function doSearch(page) {
    if (page === undefined) page = 1;

    var q = document.getElementById('searchInput').value.trim();
    if (!q) return;

    searchPage = page;

    try {
        var limit = PAGE_SIZE;
        var offset = (page - 1) * limit;

        var data = await api(
            API + '/search?q=' + encodeURIComponent(q) +
            '&limit=' + limit + '&offset=' + offset
        );

        document.getElementById('searchResults').style.display = 'block';
        document.getElementById('treeView').style.display = 'none';
        document.getElementById('commentTree').style.display = 'none';
        document.getElementById('treePagination').style.display = 'none';

        var list = document.getElementById('searchList');
        list.innerHTML = '';

        var comments = data.comments || [];
        if (comments.length === 0) {
            list.innerHTML = '<div class="empty-state">No results found.</div>';
        } else {
            comments.forEach(function(c) {
                var div = document.createElement('div');
                div.className = 'comment';
                var date = new Date(c.created_at).toLocaleString();
                div.innerHTML =
                    '<div class="comment-header">' +
                    '<span class="comment-author">' + esc(c.author) + '</span>' +
                    '<span class="comment-date">' + esc(date) + '</span>' +
                    '</div>' +
                    '<div class="comment-body">' + esc(c.body) + '</div>';
                list.appendChild(div);
            });
        }

        renderPagination('searchPagination', data.total || 0, page, limit, 'doSearch');
    } catch (e) {
        showError(e.message);
    }
}

function clearSearch() {
    document.getElementById('searchInput').value = '';
    document.getElementById('searchResults').style.display = 'none';
    document.getElementById('treeView').style.display = 'block';
    document.getElementById('commentTree').style.display = 'block';
    document.getElementById('treePagination').style.display = 'flex';
    loadTree(1);
}

// --- Pagination ---

function renderPagination(containerId, total, page, pageSize, fnName) {
    var container = document.getElementById(containerId);
    var totalPages = Math.ceil((total || 0) / pageSize);

    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    var prevDisabled = page <= 1 ? 'disabled' : '';
    var nextDisabled = page >= totalPages ? 'disabled' : '';

    container.innerHTML =
        '<button class="btn btn-outline btn-sm" ' + prevDisabled +
        ' onclick="' + fnName + '(' + (page - 1) + ')">Previous</button>' +
        '<span>Page ' + page + ' / ' + totalPages + '</span>' +
        '<button class="btn btn-outline btn-sm" ' + nextDisabled +
        ' onclick="' + fnName + '(' + (page + 1) + ')">Next</button>';
}

// --- Helpers ---

function esc(s) {
    var d = document.createElement('div');
    d.appendChild(document.createTextNode(String(s)));
    return d.innerHTML;
}

// --- Init ---

document.getElementById('searchInput').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') doSearch();
});

loadTree(1);