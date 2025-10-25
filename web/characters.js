const statusEl = document.getElementById("status");
const listEl = document.getElementById("character-list");
const nextBtn = document.getElementById("next-btn");
const reanalyseBtn = document.getElementById("reanalyse-btn");
const generateAllBtn = document.getElementById("generate-all-btn");
const progressContainer = document.getElementById("progress-container");
const progressBar = document.getElementById("progress-bar");
const progressText = document.getElementById("progress-text");

let charactersData = [];
let isBusy = false;

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function setBusy(busy) {
  isBusy = busy;
  nextBtn.disabled = busy;
  reanalyseBtn.disabled = busy;
  generateAllBtn.disabled = busy;
}

function toCharacterArray(raw) {
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw.map((character) => ({
    name: character?.name ?? "",
    description: character?.description ?? "",
    imagePath: character?.imagePath ?? "",
  }));
}

function renderCharacters(characters) {
  charactersData = toCharacterArray(characters);

  listEl.innerHTML = "";
  if (!charactersData.length) {
    const empty = document.createElement("p");
    empty.textContent = "暂无角色信息，请尝试重新识别。";
    empty.className = "section-hint";
    listEl.appendChild(empty);
    return;
  }

  charactersData.forEach((character, index) => {
    const item = document.createElement("div");
    item.className = "character-item";

    const header = document.createElement("header");
    header.textContent = `角色 ${index + 1}`;
    item.appendChild(header);

    // 角色图片显示区域
    if (character.imagePath) {
      const imageContainer = document.createElement("div");
      imageContainer.className = "character-image-container";

      const image = document.createElement("img");
      image.src = character.imagePath;
      image.alt = character.name;
      image.className = "character-image";
      imageContainer.appendChild(image);

      item.appendChild(imageContainer);
    }

    const nameLabel = document.createElement("label");
    nameLabel.textContent = "角色名称";
    item.appendChild(nameLabel);

    const nameInput = document.createElement("input");
    nameInput.type = "text";
    nameInput.value = character.name;
    nameInput.placeholder = "请输入角色名称";
    nameInput.addEventListener("input", (event) => {
      charactersData[index].name = event.target.value;
    });
    item.appendChild(nameInput);

    const descLabel = document.createElement("label");
    descLabel.style.marginTop = "12px";
    descLabel.textContent = "角色描述";
    item.appendChild(descLabel);

    const descInput = document.createElement("textarea");
    descInput.value = character.description;
    descInput.placeholder = "请输入角色的关键特征描述";
    descInput.addEventListener("input", (event) => {
      charactersData[index].description = event.target.value;
    });
    item.appendChild(descInput);

    // 生成角色图片按钮
    const buttonGroup = document.createElement("div");
    buttonGroup.className = "character-button-group";

    const generateImageBtn = document.createElement("button");
    generateImageBtn.type = "button";
    generateImageBtn.className = "character-generate-btn";
    generateImageBtn.textContent = "生成角色图片";
    generateImageBtn.addEventListener("click", () => generateCharacterImage(index));
    buttonGroup.appendChild(generateImageBtn);

    item.appendChild(buttonGroup);

    listEl.appendChild(item);
  });
}

async function loadCharacters({ forceAnalyse = false } = {}) {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    setStatus(forceAnalyse ? "正在重新识别角色..." : "正在加载角色...");
    let characters = [];

    if (!forceAnalyse) {
      const response = await fetch("/api/characters");
      if (!response.ok) {
        throw new Error("读取角色信息失败");
      }
      characters = toCharacterArray(await response.json());
    }

    if (forceAnalyse || characters.length === 0) {
      characters = await analyseCharacters();
    }

    renderCharacters(characters);
    setStatus("");
  } catch (err) {
    renderCharacters([]);
    setStatus(err.message, true);
  } finally {
    setBusy(false);
  }
}

async function analyseCharacters() {
  const response = await fetch("/api/characters/extract", {
    method: "POST",
  });
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "角色识别失败");
  }
  const data = await response.json();
  return toCharacterArray(data);
}

async function saveCharacters() {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    setStatus("正在保存角色信息...");
    const response = await fetch("/api/characters", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(charactersData),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || "保存角色失败");
    }
    setStatus("角色信息已保存，正在进入场景配置...");
    window.location.href = "scenes.html";
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    setBusy(false);
  }
}

nextBtn.addEventListener("click", saveCharacters);
reanalyseBtn.addEventListener("click", () => loadCharacters({ forceAnalyse: true }));
generateAllBtn.addEventListener("click", generateAllCharacterImages);

async function generateCharacterImage(index) {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    setStatus(`正在生成角色 ${index + 1} 的图片...`);

    const response = await fetch("/api/characters/generate-image", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ index }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || "生成图片失败");
    }

    const updatedCharacter = await response.json();
    charactersData[index] = updatedCharacter;
    renderCharacters(charactersData);
    setStatus(`角色 ${index + 1} 的图片生成成功！`);
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    setBusy(false);
  }
}

async function generateAllCharacterImages() {
  if (isBusy) {
    return;
  }
  try {
    setBusy(true);
    progressContainer.style.display = "block";
    progressBar.style.width = "0%";
    progressText.textContent = "0%";
    setStatus("正在批量生成角色图片...");

    const response = await fetch("/api/characters/generate-all", {
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

    await loadCharacters();
    setStatus("所有角色图片生成完成！");
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    setBusy(false);
    setTimeout(() => {
      progressContainer.style.display = "none";
    }, 2000);
  }
}

loadCharacters();
