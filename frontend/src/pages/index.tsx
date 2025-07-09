import Head from 'next/head';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Adicionar quaisquer outras props que getStaticProps possa retornar
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common'])),
  },
});

export default function HomePage(props: InferGetStaticPropsType<typeof getStaticProps>) {
  const { t } = useTranslation('common');

  return (
    <>
      <Head>
        <title>{t('app_title', 'Phoenix GRC')}</title> {/* Usar uma chave para o título, com fallback */}
        <meta name="description" content={t('app_description', 'Phoenix GRC Platform')} />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <main className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-[#2e026d] to-[#15162c] text-white">
        <div className="container flex flex-col items-center justify-center gap-12 px-4 py-16 ">
          <h1 className="text-5xl font-extrabold tracking-tight sm:text-[5rem]">
            {t('main_title_part1', 'Phoenix')} <span className="text-[hsl(280,100%,70%)]">{t('main_title_part2', 'GRC')}</span>
          </h1>
          <p className="text-2xl">
            {t('greeting')}
          </p>
        </div>
      </main>
    </>
  );
}
