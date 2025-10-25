const statusEl = document.getElementById("status");
const listEl = document.getElementById("scene-list");
const saveBtn = document.getElementById("save-btn");
const reanalyseBtn = document.getElementById("reanalyse-btn");

let scenesData = [];
let isBusy = false;

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function setBusy(busy) {
  isBusy = busy;
  saveBtn.disabled = busy;
  reanalyseBtn.disabled = busy;
}

function toStringArray(value) {
  if (Array.isArray(value)) {
    return value.map((item) => (typeof item === "string" ? item.trim() : "")).filter(Boolean);
  }
  if (typeof value === "string") {
    return value
      .split(/[\n,，、；;]+/)
      .map((item) => item.trim())
      .filter(Boolean);
  }
  return [];
}

function normalizeScene(scene = {}) {
  return {
    title: (scene.title ?? "").trim(),
    characters: toStringArray(scene.characters),
    description: (scene.description ?? "").trim(),
    dialogues: toStringArray(scene.dialogues),
    narration: (scene.narration ?? "").trim(),
    imagePath:
      typeof scene.imagePath === "string" ? scene.imagePath.trim() : "",
    audioPath:
      typeof scene.audioPath === "string" ? scene.audioPath.trim() : "",
  };
}

function renderScenes(scenes) {
  scenesData = scenes.map((scene) => normalizeScene(scene));
  listEl.innerHTML = "";

  if (!scenesData.length) {
    const empty = document.createElement("p");
    empty.textContent = "暂无场景信息，请尝试重新识别。";
    empty.className = "section-hint";
    listEl.appendChild(empty);
    return;
  }

  scenesData.forEach((scene, index) => {
    const item = document.createElement("section");
    item.className = "scene-item";

    const header = document.createElement("header");
    header.textContent = `场景 ${index + 1}`;
    item.appendChild(header);

    const titleLabel = document.createElement("label");
    titleLabel.textContent = "场景标题";
    item.appendChild(titleLabel);

    const titleInput = document.createElement("input");
    titleInput.type = "text";
    titleInput.value = scene.title;
    titleInput.placeholder = "请输入场景名称";
    titleInput.addEventListener("input", (event) => {
      scenesData[index].title = event.target.value;
    });
    item.appendChild(titleInput);

    const charactersLabel = document.createElement("label");
    charactersLabel.textContent = "出场人物";
    charactersLabel.style.marginTop = "12px";
    item.appendChild(charactersLabel);

    const charactersInput = document.createElement("textarea");
    charactersInput.value = scene.characters.join("\n");
    charactersInput.placeholder = "每行填写一个人物名称";
    charactersInput.addEventListener("input", (event) => {
      scenesData[index].characters = toStringArray(event.target.value);
    });
    item.appendChild(charactersInput);

    const descriptionLabel = document.createElement("label");
    descriptionLabel.textContent = "场景描述";
    descriptionLabel.style.marginTop = "12px";
    item.appendChild(descriptionLabel);

    const descriptionInput = document.createElement("textarea");
    descriptionInput.value = scene.description;
    descriptionInput.placeholder = "描述场景的视觉效果、动作、情绪等";
    descriptionInput.addEventListener("input", (event) => {
      scenesData[index].description = event.target.value;
    });
    item.appendChild(descriptionInput);

    const dialoguesLabel = document.createElement("label");
    dialoguesLabel.textContent = "关键对话";
    dialoguesLabel.style.marginTop = "12px";
    item.appendChild(dialoguesLabel);

    const dialoguesInput = document.createElement("textarea");
    dialoguesInput.value = scene.dialogues.join("\n");
    dialoguesInput.placeholder = "每行一条对话";
    dialoguesInput.addEventListener("input", (event) => {
      scenesData[index].dialogues = toStringArray(event.target.value);
    });
    item.appendChild(dialoguesInput);

    const narrationLabel = document.createElement("label");
    narrationLabel.textContent = "解说词";
    narrationLabel.style.marginTop = "12px";
    item.appendChild(narrationLabel);

    const narrationInput = document.createElement("textarea");
    narrationInput.value = scene.narration;
    narrationInput.placeholder = "旁白或解说词内容";
    narrationInput.addEventListener("input", (event) => {
      scenesData[index].narration = event.target.value;
    });
    item.appendChild(narrationInput);

    const buttonGroup = document.createElement("div");
    buttonGroup.className = "scene-button-group";

    const generateBtn = document.createElement("button");
    generateBtn.type = "button";
    generateBtn.className = "scene-generate";
    generateBtn.textContent = "生成";
    generateBtn.title = scene.imagePath
      ? "重新生成场景图片"
      : "生成场景图片";
    generateBtn.dataset.sceneIndex = String(index);
    generateBtn.disabled = !scene.description;
    generateBtn.addEventListener("click", () => handleGenerateScene(index));
    buttonGroup.appendChild(generateBtn);

    item.appendChild(buttonGroup);
    listEl.appendChild(item);
  });
}

async function loadScenes({ forceAnalyse = false } = {}) {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    setStatus(forceAnalyse ? "正在重新识别场景..." : "正在加载场景...");
    let scenes = [];

    if (!forceAnalyse) {
      const response = await fetch("/api/scenes");
      if (!response.ok) {
        throw new Error("读取场景信息失败");
      }
      scenes = await response.json();
    }

    if (forceAnalyse || scenes.length === 0) {
      scenes = await analyseScenes();
    }

    renderScenes(scenes);
    setStatus("");
  } catch (err) {
    renderScenes([]);
    setStatus(err.message, true);
  } finally {
    setBusy(false);
  }
}

async function analyseScenes() {
  const response = await fetch("/api/scenes/extract", {
    method: "POST",
  });
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "场景识别失败");
  }
  return response.json();
}

async function saveScenes() {
  console.log("saveScenes 函数被调用");
  if (isBusy) {
    console.log("当前正忙，跳过保存操作");
    return;
  }
  try {
    setBusy(true);
    setStatus("正在保存场景信息...");

    const normalizedScenes = scenesData.map(normalizeScene);
    console.log("准备保存的场景数据:", normalizedScenes);

    const response = await fetch("/api/scenes", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(normalizedScenes),
    });

    console.log("保存响应状态:", response.status);

    if (!response.ok) {
      const message = await response.text();
      console.error("保存失败:", message);
      throw new Error(message || "保存场景失败");
    }

    console.log("保存成功，准备跳转到播放页面");
    setStatus("场景信息已保存，正在跳转到播放页面...");

    // 延迟跳转，让用户看到保存成功的提示
    setTimeout(() => {
      console.log("正在跳转到 playback.html");
      window.location.href = "playback.html";
    }, 500);
  } catch (err) {
    console.error("保存场景时出错:", err);
    setStatus(err.message, true);
    setBusy(false);
  }
}

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

async function handleGenerateScene(index) {
  if (isBusy) {
    return;
  }
  const button = document.querySelector(
    `.scene-generate[data-scene-index="${index}"]`,
  );
  const previousDisabled = button?.disabled ?? false;
  if (button) {
    button.disabled = true;
  }

  let navigated = false;
  let eventSource = null;
  const taskId = `image-${index}-${Date.now()}`;
  
  try {
    setBusy(true);
    setStatus(`正在准备生成场景 ${index + 1} 的图片...`);
    
    eventSource = subscribeToProgress(
      taskId,
      (update) => {
        setStatus(update.message);
      },
      () => {
        setStatus("图片生成完成，正在打开详情...");
      },
      (error) => {
        setStatus(`生成失败: ${error}`, true);
      }
    );
    
    const response = await fetch("/api/scenes/generate-image", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ index, taskId }),
    });
    if (!response.ok) {
      const message = await response.text();
      throw new Error(message || "生成场景图片失败");
    }
    const result = await response.json();
    scenesData[index] = normalizeScene(result.scene);
    setStatus("图片生成完成，正在打开详情...");
    navigated = true;
    setBusy(false);
    window.location.href = `scene_detail.html?index=${index}`;
    return;
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    if (eventSource) {
      eventSource.close();
    }
    if (!navigated) {
      if (button) {
        button.disabled = previousDisabled;
      }
      setBusy(false);
    }
  }
}

saveBtn.addEventListener("click", saveScenes);
reanalyseBtn.addEventListener("click", () => loadScenes({ forceAnalyse: true }));

loadScenes();
