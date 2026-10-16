/* Melhoria progressiva. Não oculta conteúdo e não depende de rede. */
(() => {
  "use strict";
  const root = document.documentElement;
  const menu = document.querySelector(".site-nav");
  const menuButton = document.querySelector(".menu-toggle");
  const motionButton = document.querySelector(".motion-toggle");
  const status = document.getElementById("motion-status");
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");
  const compact = window.matchMedia("(max-width: 900px)");
  const paragraphs = [...document.querySelectorAll("[data-reveal]")];
  const art = document.querySelector("[data-parallax]");
  const hero = document.querySelector(".hero");
  const removers = [];
  const groups = [];
  let staticMode = false;
  let frame = 0;
  let disposed = false;

  function on(target, event, handler, options) {
    target.addEventListener(event, handler, options);
    removers.push(() => target.removeEventListener(event, handler, options));
  }
  function allowMotion() { return !reduce.matches && !staticMode; }
  function closeMenu(returnFocus = false) {
    if (!menu || !menuButton) return;
    const wasOpen = menu.dataset.open === "true";
    menu.dataset.open = "false";
    menuButton.setAttribute("aria-expanded", "false");
    if (returnFocus && wasOpen) menuButton.focus();
  }
  function splitWords(element) {
    const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
    const texts = [];
    while (walker.nextNode()) {
      const node = walker.currentNode;
      if (node.nodeValue.trim() && !node.parentElement.closest("svg, script, style, [data-no-reveal]")) texts.push(node);
    }
    const words = [];
    for (const text of texts) {
      const fragment = document.createDocumentFragment();
      for (const part of text.nodeValue.split(/(\s+)/u)) {
        if (!part) continue;
        if (/^\s+$/u.test(part)) fragment.append(document.createTextNode(part));
        else {
          const word = document.createElement("span");
          word.className = "reveal-word";
          word.textContent = part;
          words.push(word);
          fragment.append(word);
        }
      }
      text.replaceWith(fragment);
    }
    return words;
  }
  function requestRender() {
    if (!disposed && !frame) frame = requestAnimationFrame(render);
  }
  function render() {
    frame = 0;
    if (disposed) return;
    const enabled = allowMotion();
    const vh = window.innerHeight;
    // Agrupar leitura de layout antes de alterar classes e estilos.
    const measurements = groups.map(group => ({ group, rect: group.element.getBoundingClientRect() }));
    const heroRect = hero ? hero.getBoundingClientRect() : null;
    root.classList.toggle("motion-on", enabled);
    for (const { group, rect } of measurements) {
      const progress = enabled ? Math.min(1, Math.max(0, (0.82 * vh - rect.top) / (rect.height + 0.40 * vh))) : 1;
      const last = Math.max(1, group.words.length - 1);
      group.words.forEach((word, index) => word.classList.toggle("is-read", progress >= index / last));
    }
    if (art && heroRect) {
      const progress = Math.min(1, Math.max(0, -heroRect.top / Math.max(1, heroRect.height)));
      const displacement = enabled && !compact.matches ? progress * 24 : 0;
      art.style.setProperty("--parallax-y", `${displacement.toFixed(2)}px`);
    }
  }
  function updatePolicy() {
    const enabled = allowMotion();
    if (motionButton) {
      motionButton.disabled = reduce.matches;
      motionButton.setAttribute("aria-pressed", String(!enabled));
    }
    if (status) status.textContent = reduce.matches ? "Movimento reduzido pela preferência do sistema." : enabled ? "Realce de leitura ativado." : "Leitura estática ativada.";
    requestRender();
  }
  function destroy() {
    if (disposed) return;
    disposed = true;
    if (frame) cancelAnimationFrame(frame);
    removers.splice(0).forEach(remove => remove());
    root.classList.remove("motion-on");
    delete root.dataset.enhanced;
    groups.forEach(group => group.words.forEach(word => word.replaceWith(...word.childNodes)));
    if (menu) delete menu.dataset.open;
    if (menuButton) { menuButton.hidden = true; menuButton.setAttribute("aria-expanded", "false"); }
    if (motionButton) motionButton.hidden = true;
    if (art) art.style.removeProperty("--parallax-y");
    if (status) status.textContent = "Conteúdo completo em leitura estática.";
  }

  try {
    paragraphs.forEach(element => groups.push({ element, words: splitWords(element) }));
    if (menu && menuButton) {
      menuButton.hidden = false;
      on(menuButton, "click", () => {
        const open = menu.dataset.open !== "true";
        menu.dataset.open = String(open);
        menuButton.setAttribute("aria-expanded", String(open));
      });
      on(menu, "click", event => { if (event.target.closest("a")) closeMenu(); });
      on(document, "keydown", event => { if (event.key === "Escape") closeMenu(true); });
    }
    if (motionButton) {
      motionButton.hidden = false;
      on(motionButton, "click", () => { staticMode = !staticMode; updatePolicy(); });
    }
    on(window, "scroll", requestRender, { passive: true });
    on(window, "resize", requestRender, { passive: true });
    on(reduce, "change", updatePolicy);
    on(compact, "change", () => { closeMenu(); requestRender(); });
    root.dataset.enhanced = "true";
    updatePolicy();
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(requestRender).catch(() => {});
    window.chessStyle = Object.freeze({ destroy });
  } catch (error) {
    destroy();
    console.warn("A melhoria visual não foi iniciada. A página permanece estática.", error);
  }
})();
