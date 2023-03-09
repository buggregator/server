import { Meta, Story } from "@storybook/vue3";
import { HTML } from '@/mocks/mail';
import HtmlPreviewComponent from './HtmlPreview.vue';

export default {
  title: "Pages/Smtp/Parts",
  component: HtmlPreviewComponent
} as Meta<typeof HtmlPreviewComponent>;

const Template: Story = (args) => ({
  components: { HtmlPreviewComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<HtmlPreviewComponent v-bind="args">${HTML}</HtmlPreviewComponent>`,
});

export const HtmlPreview = Template.bind({});

HtmlPreview.args = {
  device: 'tablet',
};
