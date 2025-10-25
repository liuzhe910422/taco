const statusEl = document.getElementById("status");
const listEl = document.getElementById("scene-list");
const saveBtn = document.getElementById("save-btn");
const reanalyseBtn = document.getElementById("reanalyse-btn");
const generateAllBtn = document.getElementById("generate-all-btn");
const progressContainer = document.getElementById("progress-container");
const progressBar = document.getElementById("progress-bar");
const progressText = document.getElementById("progress-text");

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
  generateAllBtn.disabled = busy;
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
    header.style.display = "flex";
    header.style.alignItems = "center";
    header.style.gap = "8px";
    
    const headerText = document.createElement("span");
    headerText.textContent = `场景 ${index + 1}`;
    header.appendChild(headerText);
    
    const statusContainer = document.createElement("span");
    statusContainer.style.display = "flex";
    statusContainer.style.gap = "4px";
    statusContainer.style.marginLeft = "auto";
    
    const imageStatus = document.createElement("span");
    imageStatus.style.fontSize = "20px";
    imageStatus.title = scene.imagePath ? "图片已生成" : "图片未生成";
    imageStatus.textContent = scene.imagePath ? "🖼️" : "⬜";
    statusContainer.appendChild(imageStatus);
    
    const audioStatus = document.createElement("span");
    audioStatus.style.fontSize = "20px";
    audioStatus.title = scene.audioPath ? "音频已生成" : "音频未生成";
    audioStatus.textContent = scene.audioPath ? "🔊" : "🔇";
    statusContainer.appendChild(audioStatus);
    
    header.appendChild(statusContainer);
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

    const characterGenerateBtn = document.createElement("button");
    characterGenerateBtn.type = "button";
    characterGenerateBtn.className = "scene-character-generate";
    characterGenerateBtn.textContent = "人物生成";
    characterGenerateBtn.title = scene.imagePath
      ? "使用人物图片重新生成场景图片"
      : "使用人物图片生成场景图片";
    characterGenerateBtn.dataset.sceneIndex = String(index);
    characterGenerateBtn.disabled = !scene.description;
    characterGenerateBtn.addEventListener("click", () => handleGenerateSceneWithCharacters(index));
    buttonGroup.appendChild(characterGenerateBtn);

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
  try {
    setBusy(true);
    setStatus(`正在生成场景 ${index + 1} 的图片...`);
    const response = await fetch("/api/scenes/generate-image", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ index }),
    });
    if (!response.ok) {
      const message = await response.text();
      throw new Error(message || "生成场景图片失败");
    }
    const updatedScene = await response.json();
    scenesData[index] = normalizeScene(updatedScene);
    setStatus("图片生成完成，正在打开详情...");
    navigated = true;
    setBusy(false);
    window.location.href = `scene_detail.html?index=${index}`;
    return;
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    if (!navigated) {
      if (button) {
        button.disabled = previousDisabled;
      }
      setBusy(false);
    }
  }
}

async function handleGenerateSceneWithCharacters(index) {
  if (isBusy) {
    return;
  }
  const button = document.querySelector(
    `.scene-character-generate[data-scene-index="${index}"]`,
  );
  const previousDisabled = button?.disabled ?? false;
  if (button) {
    button.disabled = true;
  }

  let navigated = false;
  try {
    setBusy(true);
    setStatus(`正在使用人物图片生成场景 ${index + 1} 的图片...`);
    const response = await fetch("/api/scenes/generate-image-with-characters", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ index }),
    });
    if (!response.ok) {
      const message = await response.text();
      throw new Error(message || "生成场景图片失败");
    }
    const updatedScene = await response.json();
    scenesData[index] = normalizeScene(updatedScene);
    setStatus("图片生成完成，正在打开详情...");
    navigated = true;
    setBusy(false);
    window.location.href = `scene_detail.html?index=${index}`;
    return;
  } catch (err) {
    setStatus(err.message, true);
  } finally {
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
generateAllBtn.addEventListener("click", generateAllSceneImages);

async function generateAllSceneImages() {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    progressContainer.style.display = "block";
    progressBar.style.width = "0%";
    progressText.textContent = "0%";
    setStatus("正在批量生成场景图片...");

    const response = await fetch("/api/scenes/generate-all", {
      method: "POST",
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || "批量生成失败");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop();

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = JSON.parse(line.slice(6));
          const percent = Math.round((data.current / data.total) * 100);
          progressBar.style.width = percent + "%";
          progressText.textContent = `${data.current}/${data.total}`;
          setStatus(data.status);
        }
      }
    }

    await loadScenes();
    setStatus("所有场景图片生成完成！");
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    setBusy(false);
    setTimeout(() => {
      progressContainer.style.display = "none";
    }, 2000);
  }
}

loadScenes();
