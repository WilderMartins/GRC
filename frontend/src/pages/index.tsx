import Head from 'next/head';

export default function HomePage() {
  return (
    <>
      <Head>
        <title>Phoenix GRC</title>
        <meta name="description" content="Phoenix GRC Platform" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <main className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-[#2e026d] to-[#15162c] text-white">
        <div className="container flex flex-col items-center justify-center gap-12 px-4 py-16 ">
          <h1 className="text-5xl font-extrabold tracking-tight sm:text-[5rem]">
            Phoenix <span className="text-[hsl(280,100%,70%)]">GRC</span>
          </h1>
        </div>
      </main>
    </>
  );
}
