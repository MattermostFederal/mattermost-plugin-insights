// Allow side-effect imports of SCSS / CSS files. Webpack's style-loader
// injects the styles as a <style> tag at runtime; TypeScript only needs
// to know the module exists.

declare module '*.scss';
declare module '*.css';
