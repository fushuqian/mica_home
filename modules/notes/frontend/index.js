// 便签模块：演示模块自带数据与交互，宿主不介入内部实现。

const KEY = 'mica.notes.v1';
let styleInjected = false;

function injectStyle() {
  if (styleInjected) return;
  styleInjected = true;
  const style = document.createElement('style');
  style.id = 'mh-notes-style';
  style.textContent = `
    .mh-notes{display:grid;gap:16px;color:var(--mh-text)}
    .mh-notes .editor{
      display:flex;gap:10px;padding:10px;border-radius:var(--mh-radius);
      background:linear-gradient(150deg,var(--mh-card-a),var(--mh-card-b));
      border:1px solid var(--mh-glass-border);box-shadow:inset 0 1px 0 var(--mh-inset),var(--mh-shadow);
    }
    .mh-notes input{
      flex:1;min-width:0;border:0;outline:0;background:transparent;
      font:400 13.5px/1 inherit;color:var(--mh-text);padding:8px 10px;
    }
    .mh-notes input::placeholder{color:var(--mh-text-3)}
    .mh-notes button{
      border:0;cursor:pointer;padding:10px 18px;border-radius:999px;
      font:600 12.5px/1 inherit;color:var(--mh-bg);background:var(--mh-text);
      transition:transform .3s var(--mh-ease),box-shadow .3s;
    }
    .mh-notes button:hover{transform:translateY(-1px);box-shadow:var(--mh-shadow)}
    .mh-notes ul{list-style:none;margin:0;padding:0;display:grid;gap:10px}
    .mh-notes li{
      display:flex;align-items:center;gap:12px;padding:14px 16px;border-radius:14px;
      background:var(--mh-soft);border:1px solid var(--mh-line);
      font-size:13.5px;color:var(--mh-text);
      animation:rise .4s var(--mh-ease) both;
    }
    .mh-notes li .time{font:500 10.5px/1 var(--mh-font-m);color:var(--mh-text-3);flex:0 0 auto}
    .mh-notes li .text{flex:1;min-width:0;overflow-wrap:anywhere}
    .mh-notes li .del{
      border:0;background:transparent;cursor:pointer;color:var(--mh-text-3);padding:4px 6px;
      border-radius:8px;font-size:15px;line-height:1;transition:color .25s,background .25s;
    }
    .mh-notes li .del:hover{color:var(--mh-amber);background:color-mix(in srgb, var(--mh-amber) 10%, transparent)}
    .mh-notes .empty{padding:26px 18px;text-align:center;color:var(--mh-text-3);font-size:13px}
  `;
  document.head.appendChild(style);
}

function read() {
  try {
    return JSON.parse(localStorage.getItem(KEY) || '[]');
  } catch {
    return [];
  }
}

function write(list) {
  localStorage.setItem(KEY, JSON.stringify(list));
}

export function mount(el, api) {
  injectStyle();
  let list = read();

  el.innerHTML = `
    <div class="mh-notes">
      <form class="editor">
        <input type="text" placeholder="写点什么，回车保存…" maxlength="120" aria-label="便签内容">
        <button type="submit">保存</button>
      </form>
      <ul></ul>
    </div>`;

  const form = el.querySelector('form');
  const input = el.querySelector('input');
  const ul = el.querySelector('ul');

  const render = () => {
    if (!list.length) {
      ul.innerHTML = '<div class="empty">还没有便签。写一条试试。</div>';
      return;
    }
    ul.innerHTML = list
      .map(
        (n, i) => `
        <li style="animation-delay:${i * 40}ms">
          <span class="time">${n.at}</span>
          <span class="text">${escapeHTML(n.text)}</span>
          <button class="del" data-i="${i}" aria-label="删除">×</button>
        </li>`
      )
      .join('');
  };

  const onSubmit = (e) => {
    e.preventDefault();
    const text = input.value.trim();
    if (!text) return;
    const at = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
    list = [{ text, at }, ...list].slice(0, 30);
    write(list);
    input.value = '';
    render();
    api.toast('已保存到本机');
  };

  const onClick = (e) => {
    const btn = e.target.closest('.del');
    if (!btn) return;
    list.splice(Number(btn.dataset.i), 1);
    write(list);
    render();
  };

  form.addEventListener('submit', onSubmit);
  ul.addEventListener('click', onClick);
  render();

  return () => {
    form.removeEventListener('submit', onSubmit);
    ul.removeEventListener('click', onClick);
  };
}

function escapeHTML(s) {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
