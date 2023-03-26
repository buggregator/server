import { Meta, Story } from "@storybook/vue3";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

const iconNames = ((import.meta.env.STORYBOOK_ICON_SVG_NAMES as string) || '').split(',');

export default {
  title: "Components/IconSvg",
  component: IconSvg,
  argTypes: {
    name: {
      control: { type: 'select' },
      options: iconNames,
    },
  }
}as Meta<typeof IconSvg>;

const Template: Story = (args) => ({
  components: { IconSvg },
  setup() {
    return {
      args,
    };
  },
  template: '<div style="width: 50px;"><icon-svg v-bind="args" /></div>',
});

export const Default = Template.bind({});
Default.args = {
  name: "github"
};

export const AllIcons: Story = () => ({
  components: { IconSvg },
  setup() {
    return {
      names: iconNames,
    };
  },
  template: `
    <div style="display: flex; flex-wrap: wrap;">
      <figure v-for="name in names" class="flex flex-col items-center p-3 justify-between" style="width: 200px">
        <icon-svg :name="name" style="width: 50px; margin: auto;" />
        <figcaption>{{ name }}</figcaption>
      </figure>
    </div>
  `,
});

