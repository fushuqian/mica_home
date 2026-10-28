// 便签模块：演示模块自带数据与交互，宿主不介入内部实现。

const KEY = 'mica.notes.v1';
let styleInjected = false;

function injectStyle() {
  if (styleInjected) return;
  styleInjected = true;
  const style = document.createElement('style');
  style.textContent = `
    .mh-notes{display:grid;gap:16px}
    .mh-notes .editor{
      display:flex;gap:10px;padding:10px;border-radius:16px;
      background:linear-gradient(150deg,rgba(255,255,255,.92),rgba(255,255,255,.64));
      border:1px solid rgba(255,255,255,.95);box-shadow:0 16px 34px -26px rgba(10,20,31,.5);
    }
    .mh-notes input{
      flex:1;min-width:0;border:0;outline:0;background:transparent;
      font:400 13.5px/1 inherit;color:#0A141F;padding:8px 10px;
    }
    .mh-notes input::placeholder{color:#7B8899}
    .mh-notes button{
      border:0;cursor:pointer;padding:10px 18px;border-radius:999px;
      font:600 12.5px/1 inherit;color:#fff;background:#0A141F;
      transition:transform .3s cubic-bezier(.2,.8,.2,1),box-shadow .3s;
    }
    .mh-notes button:hover{transform:translateY(-1px);box-shadow:0 10px 22px -14px rgba(10,20,31,.8)}
    .mh-notes ul{list-style:none;margin:0;padding:0;display:grid;gap:10px}
    .mh-notes li{
      display:flex;align-items:center;gap:12px;padding:14px 16px;border-radius:14px;
      background:rgba(255,255,255,.68);border:1px solid rgba(10,20,31,.07);
      font-size:13.5px;color:#0A141F;
      animation:rise .4s cubic-bezier(.2,.8,.2,1) both;
    }
    .mh-notes li .time{font:500 10.5px/1 "IBM Plex Mono",monospace;color:#7B8899;flex:0 0 auto}
    .mh-notes li .text{flex:1;min-width:0;overflow-wrap:anywhere}
    .mh-notes li .del{
      border:0;background:transparent;cursor:pointer;color:#7B8899;padding:4px 6px;
      border-radius:8px;font-size:15px;line-height:1;transition:color .25s,background .25s;
    }
    .mh-notes li .del:hover{color:#E86A2C;background:rgba(232,106,44,.1)}
    .mh-notes .empty{padding:26px 18px;text-align:center;color:#7B8899;font-size:13px}
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
