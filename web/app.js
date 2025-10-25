const uploadArea = document.getElementById("upload-area");
const uploadLabel = document.getElementById("upload-label");
const novelInput = document.getElementById("novel-input");
const form = document.getElementById("config-form");
const statusEl = document.getElementById("status");
const saveBtn = document.getElementById("save-btn");

const llmModel = document.getElementById("llm-model");
const llmBaseUrl = document.getElementById("llm-base-url");
const llmApiKey = document.getElementById("llm-api-key");
const imageModel = document.getElementById("image-model");
const imageBaseUrl = document.getElementById("image-base-url");
const imageApiKey = document.getElementById("image-api-key");
const voiceModel = document.getElementById("voice-model");
const voiceBaseUrl = document.getElementById("voice-base-url");
const voiceApiKey = document.getElementById("voice-api-key");
const voiceSpeaker = document.getElementById("voice-speaker");
const voiceLanguage = document.getElementById("voice-language");
const videoModel = document.getElementById("video-model");
const characterCount = document.getElementById("character-count");
const sceneCount = document.getElementById("scene-count");
const animeStyle = document.getElementById("anime-style");

let currentFilePath = "";

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function setUploadLabel(filePath) {
  if (!filePath) {
    uploadLabel.textContent = "添加文件";
    currentFilePath = "";
    return;
  }
  const parts = filePath.split(/[\\/]/);
  uploadLabel.textContent = `已选择：${parts[parts.length - 1]}`;
  currentFilePath = filePath;
}

async function loadConfig() {
  try {
    const response = await fetch("/api/config");
    if (!response.ok) {
      throw new Error(`加载配置失败 (${response.status})`);
    }
    const data = await response.json();
    llmModel.value = data.llm?.model ?? data.llmModel ?? "";
    llmBaseUrl.value = data.llm?.baseUrl ?? data.llmBaseUrl ?? "";
    llmApiKey.value = data.llm?.apiKey ?? data.llmApiKey ?? "";

    const imageCfg = data.image ?? {};
    imageModel.value = imageCfg.model ?? data.imageModel ?? "";
    imageBaseUrl.value = imageCfg.baseUrl ?? data.imageBaseUrl ?? llmBaseUrl.value ?? "";
    imageApiKey.value = imageCfg.apiKey ?? data.imageApiKey ?? llmApiKey.value ?? "";

    const voiceCfg = data.voice ?? {};
    voiceModel.value = voiceCfg.model ?? "";
    voiceBaseUrl.value = voiceCfg.baseUrl ?? "";
    voiceApiKey.value = voiceCfg.apiKey ?? "";
    voiceSpeaker.value = voiceCfg.voice ?? "";
    voiceLanguage.value = voiceCfg.language ?? "";
    videoModel.value = data.videoModel ?? "";
    characterCount.value = data.characterCount ?? 0;
    sceneCount.value = data.sceneCount ?? 0;
    animeStyle.value = data.animeStyle ?? "";
    setUploadLabel(data.novelFile ?? "");
    setStatus("");
  } catch (err) {
    setStatus(err.message, true);
  }
}

async function uploadFile(file) {
  const formData = new FormData();
  formData.append("novel", file);
  setStatus("文件上传中...");
  try {
    const response = await fetch("/api/upload", {
      method: "POST",
      body: formData,
    });
    if (!response.ok) {
      throw new Error("文件上传失败");
    }
    const data = await response.json();
    setUploadLabel(data.filePath);
    setStatus("文件上传成功");
  } catch (err) {
    setStatus(err.message, true);
  }
}

uploadArea.addEventListener("click", () => novelInput.click());

novelInput.addEventListener("change", () => {
  const file = novelInput.files?.[0];
  if (file) {
    uploadFile(file);
  }
});

uploadArea.addEventListener("dragover", (event) => {
  event.preventDefault();
  uploadArea.classList.add("drag-over");
});

uploadArea.addEventListener("dragleave", () => {
  uploadArea.classList.remove("drag-over");
});

uploadArea.addEventListener("drop", (event) => {
  event.preventDefault();
  uploadArea.classList.remove("drag-over");
  const file = event.dataTransfer.files?.[0];
  if (file) {
    uploadFile(file);
  }
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  saveBtn.disabled = true;
  setStatus("保存配置中...");
  const payload = {
    novelFile: currentFilePath,
    llm: {
      model: llmModel.value.trim(),
      baseUrl: llmBaseUrl.value.trim(),
      apiKey: llmApiKey.value.trim(),
    },
    image: {
      model: imageModel.value.trim(),
      baseUrl: imageBaseUrl.value.trim(),
      apiKey: imageApiKey.value.trim(),
    },
    voice: {
      model: voiceModel.value.trim(),
      baseUrl: voiceBaseUrl.value.trim(),
      apiKey: voiceApiKey.value.trim(),
      voice: voiceSpeaker.value.trim(),
      language: voiceLanguage.value.trim(),
    },
    videoModel: videoModel.value.trim(),
    characterCount: Number(characterCount.value || 0),
    sceneCount: Number(sceneCount.value || 0),
    animeStyle: animeStyle.value.trim(),
  };

  try {
    const response = await fetch("/api/config", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || "保存配置失败");
    }
    await response.json();
    setStatus("配置已保存，正在前往角色生成...");
    window.location.href = "characters.html";
  } catch (err) {
    setStatus(err.message, true);
  } finally {
    saveBtn.disabled = false;
  }
});

loadConfig();
