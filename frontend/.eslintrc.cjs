module.exports = {
  root: true,
  env: { browser: true, es2022: true, node: true },
  extends: [
    'plugin:vue/vue3-recommended',
    'eslint:recommended',
    '@vue/eslint-config-typescript',
  ],
  parserOptions: { ecmaVersion: 'latest', sourceType: 'module' },
  rules: {
    // 允许单词组件名（如 Masthead）
    'vue/multi-word-component-names': 'off',
  },
  overrides: [
    {
      // RAW/HEX/BODY 视图把原始字节渲染在 <pre> 里，其中每个空格/换行都有意义（字节保真）。
      // 关掉会重排元素内容/缩进的格式规则，避免 eslint --fix 往 <pre> 注入空白破坏保真。
      files: [
        'src/components/detail/RawView.vue',
        'src/components/detail/BodyView.vue',
        'src/components/detail/HexView.vue',
      ],
      rules: {
        // 以下全是“可自动修复的排版规则”，eslint --fix 会据此重排 markup，
        // 从而往 <pre> 里注入/删除空白与换行（实测会把 <template>…换行…</template>
        // 压成自闭合标签、把 > 移行），破坏字节保真。对这三个文件一律关闭。
        'vue/html-indent': 'off',
        'vue/singleline-html-element-content-newline': 'off',
        'vue/multiline-html-element-content-newline': 'off',
        'vue/html-self-closing': 'off',
        'vue/max-attributes-per-line': 'off',
        'vue/html-closing-bracket-newline': 'off',
        'vue/html-closing-bracket-spacing': 'off',
        'vue/first-attribute-linebreak': 'off',
        'vue/attributes-order': 'off',
      },
    },
  ],
}
