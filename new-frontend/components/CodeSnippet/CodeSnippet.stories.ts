import { Meta, Story } from "@storybook/vue3";
import { EVENT_TYPES } from "~/config/constants";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import {HTML} from '@/mocks/mail';

export default {
  title: "Components/CodeSnippet",
  component: CodeSnippet,
} as Meta<typeof CodeSnippet>;

const Template: Story = (args) => ({
  components: { CodeSnippet },
  setup() {
    return {
      args,
    };
  },
  template: `<code-snippet v-bind="args" />`,
});

export const Object = Template.bind({});
Object.args = {
  code: {
    id: 'da076402-6f98-4ada-bae2-d77d405cf427',
    type: EVENT_TYPES.MONOLOG,
    serverName: "My server",
    origin: {
      one: 1,
      two: 2,
    },
    date: new Date(1673266869 * 1000),
    labels: ['Monolog', '200' ]
  },
  language: 'javascript'
};

export const HTMLString = Template.bind({});
HTMLString.args = {
  code: HTML,
  language: 'html'
};

export const PHPString = Template.bind({});
PHPString.args = {
  code: `use RoadRunner\\Centrifugo\\CentrifugoApiInterface;

final class UserBanService 
{
    public function __construct(
        private readonly UserRepository $repository,
        private readonly CentrifugoApiInterface $ws
    ) {}

    public function handle(string $userUuid): void
    {
        $user = $this->repository->findByPK($userUuid);

        // Ban user...

        // Disconnect from webscoket server
        $this->ws->disconnect($user->getId());
    }
}
`,
  language: 'php'
};
