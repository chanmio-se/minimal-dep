// HTMX requests carry the CSRF token in X-CSRF-Token. The token is rendered
// by the server into <meta name="csrf-token">, so the cookie can stay HttpOnly.
document.addEventListener("htmx:configRequest", function (event) {
  var meta = document.querySelector('meta[name="csrf-token"]');
  if (!meta || !meta.content) {
    return;
  }
  event.detail.headers["X-CSRF-Token"] = meta.content;
});
