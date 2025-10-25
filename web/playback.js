const playbackContent = document.getElementById("playback-content");
const prevBtn = document.getElementById("prev-btn");
const nextBtn = document.getElementById("next-btn");
const pauseBtn = document.getElementById("pause-btn");
const closeBtn = document.getElementById("close-btn");

let scenes = [];
let currentIndex = 0;
let audioElement = null;
let isPaused = false;

// 加载场景数据
async function loadScenes() {
  try {
    const response = await fetch("/api/scenes");
    if (!response.ok) {
      throw new Error("加载场景失败");
    }
    scenes = await response.json();

    if (scenes.length === 0) {
      showPlaceholder("暂无场景数据，请先创建场景。");
      return;
    }

    // 获取 URL 参数中的起始索引
    const urlParams = new URLSearchParams(window.location.search);
    const startIndex = parseInt(urlParams.get("index") || "0", 10);
    currentIndex = Math.max(0, Math.min(startIndex, scenes.length - 1));

    renderScene(currentIndex);
  } catch (error) {
    console.error(error);
    showPlaceholder("加载场景失败: " + error.message);
  }
}

// 显示占位符
function showPlaceholder(message) {
  playbackContent.innerHTML = `<p class="playback-placeholder">${message}</p>`;
  prevBtn.disabled = true;
  nextBtn.disabled = true;
  pauseBtn.disabled = true;
}

// 渲染场景
function renderScene(index) {
  if (index < 0 || index >= scenes.length) {
    showPlaceholder("没有更多场景了");
    return;
  }

  const scene = scenes[index];
  currentIndex = index;

  // 构建场景 HTML
  let html = `
    <div class="playback-progress">第 ${index + 1} 个场景，共 ${scenes.length} 个场景</div>
    <h1 class="playback-title">${escapeHtml(scene.title || "未命名场景")}</h1>
  `;

  // 显示图片
  if (scene.imagePath) {
    html += `
      <div class="playback-image-container">
        <img src="${escapeHtml(scene.imagePath)}" alt="${escapeHtml(scene.title)}" class="playback-image">
      </div>
    `;
  }

  // 显示解说词
  if (scene.narration && scene.narration.trim()) {
    html += `
      <div class="playback-narration">${escapeHtml(scene.narration)}</div>
    `;
  }

  // 显示音频或提示
  if (scene.audioPath && scene.audioPath.trim()) {
    html += `
      <div class="playback-audio-container">
        <audio id="scene-audio" class="playback-audio" controls autoplay>
          <source src="${escapeHtml(scene.audioPath)}" type="audio/mpeg">
          您的浏览器不支持音频播放。
        </audio>
      </div>
    `;
  } else {
    html += `
      <div class="playback-narration" style="background: #fff3cd; border-color: #ffc107; color: #856404;">
        ⚠️ 此场景暂无音频，将在 3 秒后自动切换到下一个场景
      </div>
    `;
  }

  playbackContent.innerHTML = html;

  // 更新按钮状态
  prevBtn.disabled = index === 0;
  nextBtn.disabled = index === scenes.length - 1;
  pauseBtn.disabled = !scene.audioPath || !scene.audioPath.trim();
  pauseBtn.textContent = "暂停";
  isPaused = false;

  // 设置音频事件监听
  if (scene.audioPath && scene.audioPath.trim()) {
    audioElement = document.getElementById("scene-audio");
    if (audioElement) {
      // 音频播放结束后自动播放下一个场景
      audioElement.addEventListener("ended", () => {
        if (currentIndex < scenes.length - 1 && !isPaused) {
          setTimeout(() => {
            renderScene(currentIndex + 1);
          }, 1000); // 延迟1秒后播放下一个场景
        } else if (currentIndex === scenes.length - 1) {
          showCompletionMessage();
        }
      });

      // 音频播放失败处理
      audioElement.addEventListener("error", (e) => {
        console.error("音频加载失败:", e);
      });
    }
  } else {
    // 没有音频时，3秒后自动切换到下一个场景
    audioElement = null;
    if (currentIndex < scenes.length - 1) {
      setTimeout(() => {
        if (currentIndex === index) { // 确保用户没有手动切换场景
          renderScene(currentIndex + 1);
        }
      }, 3000);
    } else {
      // 最后一个场景，延迟后显示完成消息
      setTimeout(() => {
        if (currentIndex === index) {
          showCompletionMessage();
        }
      }, 3000);
    }
  }
}

// 显示完成消息
function showCompletionMessage() {
  playbackContent.innerHTML = `
    <div class="playback-placeholder">
      <h2 style="color: #688bff; margin-bottom: 16px;">🎉 所有场景播放完毕！</h2>
      <p>您已经观看完所有 ${scenes.length} 个场景。</p>
    </div>
  `;
  prevBtn.disabled = false;
  nextBtn.disabled = true;
  pauseBtn.disabled = true;
}

// HTML 转义，防止 XSS
function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// 上一个场景
function handlePrev() {
  if (currentIndex > 0) {
    renderScene(currentIndex - 1);
  }
}

// 下一个场景
function handleNext() {
  if (currentIndex < scenes.length - 1) {
    renderScene(currentIndex + 1);
  }
}

// 暂停/继续
function handlePause() {
  if (!audioElement) {
    return;
  }

  if (isPaused) {
    audioElement.play();
    pauseBtn.textContent = "暂停";
    isPaused = false;
  } else {
    audioElement.pause();
    pauseBtn.textContent = "继续";
    isPaused = true;
  }
}

// 关闭页面
function handleClose() {
  window.location.href = "scenes.html";
}

// 绑定事件
prevBtn.addEventListener("click", handlePrev);
nextBtn.addEventListener("click", handleNext);
pauseBtn.addEventListener("click", handlePause);
closeBtn.addEventListener("click", handleClose);

// 键盘快捷键
document.addEventListener("keydown", (e) => {
  if (e.key === "ArrowLeft" && !prevBtn.disabled) {
    handlePrev();
  } else if (e.key === "ArrowRight" && !nextBtn.disabled) {
    handleNext();
  } else if (e.key === " " && !pauseBtn.disabled) {
    e.preventDefault();
    handlePause();
  }
});

// 初始化
loadScenes();
