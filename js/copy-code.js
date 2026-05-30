(function () {
  var clipboardIcon =
    '<svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">' +
    '<path d="M8 7a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2h-8a2 2 0 0 1-2-2V7Z"></path>' +
    '<path d="M6 15H5a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>' +
    "</svg>";

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text);
    }

    return new Promise(function (resolve, reject) {
      var textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.setAttribute("readonly", "");
      textarea.className = "copy-code-buffer";
      document.body.appendChild(textarea);
      textarea.select();

      try {
        document.execCommand("copy");
        resolve();
      } catch (error) {
        reject(error);
      } finally {
        document.body.removeChild(textarea);
      }
    });
  }

  function setButtonState(button, copied) {
    var label = button.querySelector(".copy-code-label");
    button.classList.toggle("is-copied", copied);
    button.setAttribute("aria-label", copied ? "Copied code" : "Copy code");
    label.textContent = copied ? "Copied" : "Copy";
  }

  function handleCopy(wrapper) {
    var code = wrapper.querySelector("pre code");
    var button = wrapper.querySelector(".copy-code-button");
    var text = code ? code.textContent.trim() : "";

    if (!text || !button) {
      return;
    }

    copyText(text).then(function () {
      setButtonState(button, true);

      window.clearTimeout(button.copyResetTimer);
      button.copyResetTimer = window.setTimeout(function () {
        setButtonState(button, false);
      }, 1400);
    });
  }

  function wrapBlock(pre) {
    if (pre.parentElement && pre.parentElement.classList.contains("code-block")) {
      return;
    }

    var wrapper = document.createElement("div");
    wrapper.className = "code-block";

    var button = document.createElement("button");
    button.type = "button";
    button.className = "copy-code-button";
    button.setAttribute("aria-label", "Copy code");
    button.innerHTML =
      clipboardIcon + '<span class="copy-code-label">Copy</span>';

    pre.parentNode.insertBefore(wrapper, pre);
    wrapper.appendChild(button);
    wrapper.appendChild(pre);
  }

  function init() {
    document.querySelectorAll("pre").forEach(wrapBlock);

    document.addEventListener("click", function (event) {
      var button = event.target.closest(".copy-code-button");
      if (!button) {
        return;
      }

      handleCopy(button.closest(".code-block"));
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
