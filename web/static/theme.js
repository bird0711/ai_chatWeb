(function () {
  var storageKey = "ai-chat-theme";
  var root = document.documentElement;

  function storedTheme() {
    try {
      return window.localStorage.getItem(storageKey);
    } catch (error) {
      return "";
    }
  }

  function systemTheme() {
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      return "dark";
    }
    return "light";
  }

  function activeTheme() {
    return storedTheme() || systemTheme();
  }

  function applyTheme(theme) {
    root.setAttribute("data-theme", theme);
    document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
      button.textContent = theme === "dark" ? "白天模式" : "黑夜模式";
      button.setAttribute("aria-label", theme === "dark" ? "切换到白天模式" : "切换到黑夜模式");
    });
  }

  function saveTheme(theme) {
    try {
      window.localStorage.setItem(storageKey, theme);
    } catch (error) {
      return;
    }
  }

  function setupThemeToggle() {
    applyTheme(activeTheme());
    document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
      button.addEventListener("click", function () {
        var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
        saveTheme(next);
        applyTheme(next);
      });
    });
  }

  function readCookie(name) {
    var prefix = name + "=";
    var parts = document.cookie ? document.cookie.split(";") : [];
    for (var i = 0; i < parts.length; i += 1) {
      var part = parts[i].trim();
      if (part.indexOf(prefix) === 0) {
        return decodeURIComponent(part.slice(prefix.length));
      }
    }
    return "";
  }

  function setupCSRFTokens() {
    var token = readCookie("csrf_token");
    if (!token) {
      return;
    }
    document.querySelectorAll("form[method='post'], form[method='POST']").forEach(function (form) {
      var input = form.querySelector("input[name='csrf_token']");
      if (!input) {
        input = document.createElement("input");
        input.type = "hidden";
        input.name = "csrf_token";
        form.insertBefore(input, form.firstChild);
      }
      input.value = token;
    });
  }

  function setupPage() {
    setupThemeToggle();
    setupCSRFTokens();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", setupPage);
  } else {
    setupPage();
  }
})();
