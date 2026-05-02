(function () {
  function confirmForms() {
    document.querySelectorAll("form[data-confirm]").forEach(function (form) {
      form.addEventListener("submit", function (event) {
        var message = form.getAttribute("data-confirm");
        if (message && !window.confirm(message)) {
          event.preventDefault();
        }
      });
    });
  }

  function setupChat() {
    var messages = document.getElementById("messages");
    var form = document.getElementById("send-form");
    var status = document.getElementById("chat-status");
    if (!messages || !form || !status) {
      return;
    }

    var chatID = messages.getAttribute("data-chat-id");
    var roleCount = parseInt(messages.getAttribute("data-role-count") || "0", 10);
    var aiReviewEnabled = messages.getAttribute("data-ai-review-enabled") === "1";
    var submitButton = form.querySelector("button[type='submit']");
    var textarea = form.querySelector("textarea[name='content']");
    var searchInput = document.querySelector("[data-message-search]");
    var senderFilter = document.querySelector("[data-message-sender-filter]");
    var searchClear = document.querySelector("[data-message-search-clear]");
    var searchStatus = document.querySelector("[data-message-search-status]");
    var searchEmpty = messages.querySelector("[data-message-search-empty]");
    var pollTimer = null;
    var pendingSinceID = 0;
    var pendingAI = 0;
    var lastID = findLastMessageID(messages);

    form.addEventListener("submit", function (event) {
      var asyncAction = form.getAttribute("data-async-action");
      if (!window.fetch || !asyncAction) {
        return;
      }
      event.preventDefault();
      if (!textarea.value.trim()) {
        setStatus("请输入消息内容。", true);
        return;
      }

      var payload = new FormData(form);
      setBusy(true);
      setStatus("正在发送...");
      fetch(asyncAction, {
        method: "POST",
        body: payload,
        headers: {"Accept": "application/json"}
      })
        .then(function (response) {
          return response.json().then(function (body) {
            if (!response.ok) {
              throw new Error(body.error || "发送失败");
            }
            return body;
          });
        })
        .then(function (body) {
          appendMessage(body.message);
          textarea.value = "";
          pendingSinceID = body.message.id;
          pendingAI = 0;
          setStatus(aiReviewEnabled ? "AI 正在回复，随后进行互评..." : "AI 正在回复...");
          startPolling();
        })
        .catch(function (error) {
          setStatus(error.message, true);
        })
        .finally(function () {
          setBusy(false);
        });
    });

    if (textarea) {
      textarea.addEventListener("keydown", handleSendShortcut, true);
      textarea.addEventListener("keypress", handleSendShortcut, true);
    }
    setupMessageSearch();

    function handleSendShortcut(event) {
      if (!isPlainEnter(event)) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      submitSendForm();
    }

    function isPlainEnter(event) {
      var key = event.key || "";
      var code = event.code || "";
      var keyCode = event.keyCode || event.which || 0;
      var isEnter = key === "Enter" || code === "Enter" || keyCode === 13;
      if (!isEnter) {
        return false;
      }
      if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey) {
        return false;
      }
      if (event.isComposing || keyCode === 229) {
        return false;
      }
      return true;
    }

    function submitSendForm() {
      if (textarea.disabled || (submitButton && submitButton.disabled)) {
        return;
      }
      if (form.requestSubmit) {
        form.requestSubmit();
        return;
      }
      if (submitButton) {
        submitButton.click();
      }
    }

    function startPolling() {
      if (pollTimer) {
        window.clearInterval(pollTimer);
      }
      pollTimer = window.setInterval(fetchUpdates, 1000);
      fetchUpdates();
    }

    function stopPolling(message, isError) {
      if (pollTimer) {
        window.clearInterval(pollTimer);
        pollTimer = null;
      }
      setStatus(message || "", isError);
    }

    function fetchUpdates() {
      fetch("/chats/" + encodeURIComponent(chatID) + "/messages/updates?after_id=" + encodeURIComponent(String(lastID)), {
        headers: {"Accept": "application/json"}
      })
        .then(function (response) {
          return response.json().then(function (body) {
            if (!response.ok) {
              throw new Error(body.error || "获取消息失败");
            }
            return body;
          });
        })
        .then(function (body) {
          var gotSystem = false;
          (body.messages || []).forEach(function (message) {
            appendMessage(message);
            if (message.id > pendingSinceID && message.sender_type === "ai") {
              pendingAI += 1;
            }
            if (message.id > pendingSinceID && message.sender_type === "system") {
              gotSystem = true;
            }
          });
          if (pendingSinceID > 0 && expectedAIReplies() > 0 && pendingAI >= expectedAIReplies()) {
            stopPolling("");
          } else if (gotSystem) {
            stopPolling("AI 回复出现异常，请查看系统消息。", true);
          }
        })
        .catch(function (error) {
          stopPolling(error.message, true);
        });
    }

    function appendMessage(message) {
      if (!message || document.querySelector('[data-message-id="' + message.id + '"]')) {
        return;
      }
      var empty = messages.querySelector("[data-empty-messages]");
      if (empty) {
        empty.remove();
      }
      var row = renderMessage(message);
      if (searchEmpty) {
        messages.insertBefore(row, searchEmpty);
      } else {
        messages.appendChild(row);
      }
      lastID = Math.max(lastID, message.id);
      applyMessageSearch();
      if (!row.hidden) {
        row.scrollIntoView({block: "nearest"});
      }
    }

    function renderMessage(message) {
      var article = document.createElement("article");
      article.className = "message-row " + message.sender_type;
      article.setAttribute("data-message-id", String(message.id));
      article.setAttribute("data-sender-type", message.sender_type);

      if (message.sender_type !== "user") {
        article.appendChild(renderAvatar(message.sender_avatar, message.sender_name));
      }

      var bubble = document.createElement("div");
      bubble.className = "message-bubble";

      var meta = document.createElement("div");
      meta.className = "message-meta";

      var name = document.createElement("strong");
      name.textContent = message.sender_name;
      var label = document.createElement("span");
      label.textContent = senderLabel(message.sender_type);
      meta.appendChild(name);
      meta.appendChild(label);

      var content = document.createElement("p");
      content.className = "message-content";
      content.textContent = message.content;

      bubble.appendChild(meta);
      bubble.appendChild(content);
      article.appendChild(bubble);
      if (message.sender_type === "user") {
        article.appendChild(renderAvatar("", "我"));
      }
      return article;
    }

    function setupMessageSearch() {
      if (!searchInput || !senderFilter) {
        return;
      }
      searchInput.addEventListener("input", applyMessageSearch);
      senderFilter.addEventListener("change", applyMessageSearch);
      if (searchClear) {
        searchClear.addEventListener("click", function () {
          searchInput.value = "";
          senderFilter.value = "";
          applyMessageSearch();
          searchInput.focus();
        });
      }
      applyMessageSearch();
    }

    function applyMessageSearch() {
      if (!searchInput || !senderFilter) {
        return;
      }
      var query = normalizeSearch(searchInput.value);
      var sender = senderFilter.value;
      var rows = Array.from(messages.querySelectorAll(".message-row"));
      var matched = 0;

      rows.forEach(function (row) {
        var rowSender = row.getAttribute("data-sender-type") || senderTypeFromClass(row);
        var senderMatches = !sender || rowSender === sender;
        var textMatches = !query || normalizeSearch(row.textContent).indexOf(query) !== -1;
        var visible = senderMatches && textMatches;
        row.hidden = !visible;
        row.classList.toggle("filtered-out", !visible);
        if (visible) {
          matched += 1;
        }
      });

      var filtering = Boolean(query || sender);
      if (searchEmpty) {
        searchEmpty.hidden = !filtering || matched > 0 || rows.length === 0;
      }
      if (searchStatus) {
        if (!filtering) {
          searchStatus.textContent = "输入关键词或选择发送者，筛选当前已加载的历史消息。";
        } else if (matched === 0) {
          searchStatus.textContent = "没有匹配的消息。";
        } else {
          searchStatus.textContent = "已筛选出 " + matched + " 条消息。";
        }
      }
    }

    function renderAvatar(value, fallback) {
      if (isImageAvatar(value)) {
        var image = document.createElement("img");
        image.className = "message-avatar avatar-image";
        image.src = value.trim();
        image.alt = fallback || "AI";
        return image;
      }
      var avatar = document.createElement("span");
      avatar.className = "message-avatar";
      avatar.textContent = avatarText(value || fallback);
      return avatar;
    }

    function expectedAIReplies() {
      if (!aiReviewEnabled) {
        return roleCount;
      }
      return roleCount + Math.min(2, roleCount);
    }

    function setBusy(busy) {
      if (submitButton) {
        submitButton.disabled = busy;
      }
      if (textarea) {
        textarea.disabled = busy;
      }
    }

    function setStatus(message, isError) {
      status.textContent = message || "";
      status.classList.toggle("error", Boolean(isError));
    }
  }

  function findLastMessageID(container) {
    var maxID = 0;
    container.querySelectorAll("[data-message-id]").forEach(function (node) {
      var id = parseInt(node.getAttribute("data-message-id") || "0", 10);
      if (id > maxID) {
        maxID = id;
      }
    });
    return maxID;
  }

  function senderLabel(type) {
    if (type === "user") {
      return "用户";
    }
    if (type === "system") {
      return "系统";
    }
    return "AI";
  }

  function senderTypeFromClass(row) {
    if (row.classList.contains("user")) {
      return "user";
    }
    if (row.classList.contains("system")) {
      return "system";
    }
    return "ai";
  }

  function normalizeSearch(value) {
    return (value || "").toLocaleLowerCase().trim();
  }

  function avatarText(value) {
    value = (value || "").trim();
    if (!value) {
      return "AI";
    }
    return Array.from(value).slice(0, 2).join("");
  }

  function isImageAvatar(value) {
    value = (value || "").trim().toLowerCase();
    return value.indexOf("/uploads/") === 0 || value.indexOf("/static/") === 0;
  }

  confirmForms();
  setupChat();
})();
