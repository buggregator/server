const path = require('path');
const postcss = require('postcss');

module.exports = {
  stories: [
    "../stories/**/*.stories.mdx",
    "../stories/**/*.stories.@(js|jsx|ts|tsx)",
    "../components/**/*.stories.@(js|jsx|ts|tsx)",
  ],
  addons: [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/addon-interactions",
    {
      name: "@storybook/addon-postcss",
      options: {
        cssLoaderOptions: {
          importLoaders: 1,
        },
        postcssLoaderOptions: {
          implementation: postcss,
        },
      },
    },
  ],
  core: {
    builder: "@storybook/builder-vite",
  },
  framework: "@storybook/vue3",
  async viteFinal(config) {
    config.resolve.alias = {
      ...config.resolve.alias,
      '@': path.resolve(__dirname, "../"),
      '~': path.resolve(__dirname, "../"),
    };

    return {
      ...config,
      define: {
        ...config.define,
        global: "window",
      },
    };
  },
};
