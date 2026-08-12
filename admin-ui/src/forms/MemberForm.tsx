import { IconInfoCircle } from "@tabler/icons-react";
import { Form, Formik } from "formik";
import type { FC } from "react";
import {
  Card,
  FieldSet,
  Input,
  Keywords,
  LinkListField,
  SubmitButton,
  Switch,
  Textarea,
} from "@/components/ui";
import { parseExternalLinks } from "@/forms/GameForm";
import { useI18n } from "@/lib/stores/i18n";
import { memberValidationSchema } from "@/lib/validation/member";

export const emptyMemberFormValues: MemberFormValues = {
  name: "",
  tags: "",
  description: "",
  links: [],
  is_published: false,
};

export const toMemberFormValues = (
  member: ResponseMember,
): MemberFormValues => ({
  name: member.name,
  tags: member.tags,
  description: member.description,
  links: parseExternalLinks(member.links_json),
  is_published: member.is_published,
});

interface MemberFormProps {
  initialValues: MemberFormValues;
  submitLabel: string;
  onSubmit: (values: MemberFormValues) => Promise<void>;
}

const MemberForm: FC<MemberFormProps> = ({
  initialValues,
  submitLabel,
  onSubmit,
}) => {
  const { t } = useI18n();

  return (
    <Formik
      initialValues={initialValues}
      validationSchema={memberValidationSchema(t)}
      onSubmit={onSubmit}
    >
      <Form noValidate className="space-y-4">
        <Card>
          <Card.Header icon={IconInfoCircle}>
            <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
              {t("members.form.basics")}
            </h2>
          </Card.Header>
          <Card.Body className="space-y-4">
            <Input
              name="name"
              label={t("members.form.name")}
              placeholder={t("members.form.namePlaceholder")}
            />

            <Keywords
              name="tags"
              label={t("members.form.tags")}
              placeholder={t("members.form.tagsPlaceholder")}
              output="string"
            />

            <Textarea
              name="description"
              label={t("members.form.description")}
              rows={5}
            />

            <FieldSet
              legend={t("members.form.links")}
              description={t("members.form.linksHelp")}
            >
              <LinkListField
                name="links"
                labelPlaceholder={t("members.form.linkLabelPlaceholder")}
                urlPlaceholder={t("members.form.linkUrlPlaceholder")}
                addLabel={t("members.form.addLink")}
                emptyLabel={t("members.form.noLinks")}
                removeLabel={t("common.delete")}
              />
            </FieldSet>

            <Switch name="is_published" label={t("members.form.published")} />
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {t("members.form.orderHint")}
            </p>
          </Card.Body>
        </Card>

        <SubmitButton variant="default" loadingText={t("common.loading")}>
          {submitLabel}
        </SubmitButton>
      </Form>
    </Formik>
  );
};

export default MemberForm;
