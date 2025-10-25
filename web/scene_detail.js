const statusEl = document.getElementById("status");
const navEl = document.getElementById("detail-nav");
const imageContainer = document.getElementById("image-container");
const audioContainer = document.getElementById("audio-container");
const narrationContent = document.getElementById("narration-content");
const dialogueContent = document.getElementById("dialogue-content");
const descriptionContent = document.getElementById("description-content");
const sceneTitleEl = document.getElementById("scene-title");
const backBtn = document.getElementById("back-btn");
const closeBtn = document.getElementById("close-btn");
const generateAudioBtn = document.getElementById("generate-audio-btn");

let currentSceneIndex = null;
let currentScene = null;
let isGeneratingAudio = false;

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function getSceneIndex() {
  const params = new URLSearchParams(window.location.search);
  const indexParam = params.get("index");
  if (indexParam === null) {
    return null;
  }
  const parsed = Number.parseInt(indexParam, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    return null;
  }
  return parsed;
}

function normalizeScene(scene = {}) {
  return {
    title: (scene.title ?? "").trim(),
    characters: Array.isArray(scene.characters)
      ? scene.characters.map((item) => String(item ?? "").trim()).filter(Boolean)
      : [],
    description: (scene.description ?? "").trim(),
    dialogues: Array.isArray(scene.dialogues)
      ? scene.dialogues.map((item) => String(item ?? "").trim()).filter(Boolean)
      : [],
    narration: (scene.narration ?? "").trim(),
    imagePath:
      typeof scene.imagePath === "string" ? scene.imagePath.trim() : "",
    audioPath:
      typeof scene.audioPath === "string" ? scene.audioPath.trim() : "",
  };
}

function renderImage(scene, index) {
  imageContainer.innerHTML = "";
  if (scene.imagePath) {
    const img = document.createElement("img");
    img.src = scene.imagePath;
    img.alt = scene.title || `场景 ${index + 1}`;
    img.className = "detail-image";
    imageContainer.appendChild(img);

    if (scene.characters.length) {
      const info = document.createElement("div");
      info.className = "detail-text";
      info.style.marginTop = "16px";
      const paragraph = document.createElement("p");
      paragraph.textContent = `出场人物：${scene.characters.join("、")}`;
      info.appendChild(paragraph);
      imageContainer.appendChild(info);
    }
  } else {
    const placeholder = document.createElement("p");
    placeholder.className = "detail-placeholder";
    placeholder.textContent = "尚未生成图片，请返回上一页点击生成按钮。";
    imageContainer.appendChild(placeholder);

    if (scene.characters.length) {
      const info = document.createElement("div");
      info.className = "detail-text";
      info.style.marginTop = "16px";
      const paragraph = document.createElement("p");
      paragraph.textContent = `出场人物：${scene.characters.join("、")}`;
      info.appendChild(paragraph);
      imageContainer.appendChild(info);
    }
  }
}

function setBlockText(container, text, emptyMessage) {
  container.innerHTML = "";
  if (!text) {
    const placeholder = document.createElement("p");
    placeholder.className = "detail-placeholder";
    placeholder.textContent = emptyMessage;
    container.appendChild(placeholder);
    return;
  }
  const paragraph = document.createElement("p");
  paragraph.textContent = text;
  container.appendChild(paragraph);
}

function setListText(container, items, emptyMessage) {
  container.innerHTML = "";
  if (!items.length) {
    const placeholder = document.createElement("p");
    placeholder.className = "detail-placeholder";
    placeholder.textContent = emptyMessage;
    container.appendChild(placeholder);
    return;
  }
  items.forEach((item) => {
    const paragraph = document.createElement("p");
    paragraph.textContent = item;
    container.appendChild(paragraph);
  });
}

function setupNavigation() {
  const buttons = navEl.querySelectorAll("button[data-target]");
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      const target = button.dataset.target;
      buttons.forEach((btn) => btn.classList.toggle("active", btn === button));
      document
        .querySelectorAll(".detail-panel")
        .forEach((panel) => panel.classList.toggle("active", panel.id === target));
    });
  });
}

async function loadSceneDetail() {
  const index = getSceneIndex();
  if (index === null) {
    setStatus("缺少场景索引参数", true);
    return;
  }

  try {
    setStatus("正在加载场景详情...");
    const response = await fetch("/api/scenes");
    if (!response.ok) {
      throw new Error("读取场景数据失败");
    }
    const scenes = await response.json();
    if (index < 0 || index >= scenes.length) {
      throw new Error("场景索引超出范围");
    }

    const scene = normalizeScene(scenes[index]);
    currentScene = scene;
    currentSceneIndex = index;
    const displayTitle = scene.title || `场景 ${index + 1}`;
    sceneTitleEl.textContent = displayTitle;
    document.title = `Taco - ${displayTitle}`;

    renderImage(scene, index);
    renderAudio(scene);
    setBlockText(
      narrationContent,
      scene.narration,
      "暂无解说词内容，可返回上一页编辑后保存。",
    );
    setListText(
      dialogueContent,
      scene.dialogues,
      "暂无关键对话，可返回上一页编辑后保存。",
    );
    setBlockText(
      descriptionContent,
      scene.description,
      "暂无场景描述，可返回上一页编辑后保存。",
    );

    setStatus("");
  } catch (err) {
    setStatus(err.message, true);
  }
}

backBtn.addEventListener("click", () => {
  window.location.href = "scenes.html";
});

closeBtn.addEventListener("click", () => {
  window.location.href = "scenes.html";
});

function subscribeToProgress(taskId, onUpdate, onComplete, onError) {
  const eventSource = new EventSource(`/api/progress/${taskId}`);
  
  eventSource.onmessage = (event) => {
    try {
      const update = JSON.parse(event.data);
      if (update.error) {
        eventSource.close();
        if (onError) onError(update.error);
      } else if (update.completed) {
        eventSource.close();
        if (onComplete) onComplete(update);
      } else {
        if (onUpdate) onUpdate(update);
      }
    } catch (err) {
      console.error("解析进度更新失败:", err);
    }
  };
  
  eventSource.onerror = () => {
    eventSource.close();
    if (onError) onError("进度连接失败");
  };
  
  return eventSource;
}

if (generateAudioBtn) {
  generateAudioBtn.addEventListener("click", async (e) => {
    console.log("[生成声音] 按钮被点击");
    console.log("[生成声音] isGeneratingAudio:", isGeneratingAudio);
    console.log("[生成声音] currentSceneIndex:", currentSceneIndex);

    if (isGeneratingAudio) {
      console.log("[生成声音] 已经在生成中，跳过");
      setStatus("正在生成中，请稍候...", false);
      return;
    }

    if (currentSceneIndex === null) {
      console.log("[生成声音] 场景索引为空");
      setStatus("场景索引无效，请刷新页面重试", true);
      return;
    }

    let eventSource = null;
    const taskId = `audio-${currentSceneIndex}-${Date.now()}`;
    
    try {
      isGeneratingAudio = true;
      generateAudioBtn.disabled = true;
      const originalText = generateAudioBtn.textContent;
      generateAudioBtn.textContent = "生成中...";
      setStatus(`正在准备生成场景 ${currentSceneIndex + 1} 的声音...`);

      eventSource = subscribeToProgress(
        taskId,
        (update) => {
          setStatus(update.message);
        },
        () => {
          setStatus("语音生成完成！");
        },
        (error) => {
          setStatus(`生成失败: ${error}`, true);
        }
      );

      console.log("[生成声音] 发送 API 请求:", { index: currentSceneIndex, taskId });

      const response = await fetch("/api/scenes/generate-audio", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ index: currentSceneIndex, taskId }),
      });

      console.log("[生成声音] API 响应状态:", response.status);

      if (!response.ok) {
        const message = await response.text();
        console.error("[生成声音] API 错误:", message);
        throw new Error(message || "生成声音失败");
      }

      const result = await response.json();
      const updatedScene = normalizeScene(result.scene);
      console.log("[生成声音] 生成成功，音频路径:", updatedScene.audioPath);
      currentScene = updatedScene;
      renderAudio(updatedScene);
      setStatus("语音生成完成！");
    } catch (err) {
      console.error("[生成声音] 错误:", err);
      setStatus(`生成失败: ${err.message}`, true);
      if (generateAudioBtn) {
        generateAudioBtn.disabled = false;
      }
    } finally {
      if (eventSource) {
        eventSource.close();
      }
      if (generateAudioBtn) {
        generateAudioBtn.textContent = currentScene?.audioPath ? "重新生成声音" : "生成声音";
      }
      isGeneratingAudio = false;
      console.log("[生成声音] 流程结束");
    }
  });
  console.log("[初始化] 生成声音按钮事件监听器已绑定");
} else {
  console.error("[初始化] 未找到生成声音按钮元素");
}

function renderAudio(scene) {
  if (!audioContainer || !generateAudioBtn) {
    return;
  }

  audioContainer.innerHTML = "";

  if (scene.audioPath) {
    const audio = document.createElement("audio");
    audio.controls = true;
    audio.src = scene.audioPath;
    audioContainer.appendChild(audio);

    const hint = document.createElement("p");
    hint.className = "detail-placeholder";
    hint.textContent = "可以点击播放按钮试听效果，若需更新可重新生成。";
    audioContainer.appendChild(hint);

    if (generateAudioBtn) {
      generateAudioBtn.textContent = "重新生成声音";
      generateAudioBtn.disabled = false;
    }
  } else {
    const placeholder = document.createElement("p");
    placeholder.className = "detail-placeholder";
    placeholder.textContent = "尚未生成声音，可点击下方按钮创建语音。";
    audioContainer.appendChild(placeholder);

    if (generateAudioBtn) {
      generateAudioBtn.textContent = "生成声音";
      generateAudioBtn.disabled = false;
    }
  }
}

setupNavigation();
loadSceneDetail();
