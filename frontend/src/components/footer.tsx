import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { Poppins } from 'next/font/google';

const poppins = Poppins({ weight: '400', subsets: ['latin'] });


export const FooterSection = () => {
  return (
    <footer id="footer" className="container max-w-7xl pt-24 mt-auto xl:py-32 mx-auto z-10">
      <div className="p-10 bg-card border border-secondary xl:rounded-b-2xl  rounded-2xl rounded-b-none shadow-nav">
        <div className="flex flex-col gap-6 md:flex-row md:justify-between w-full">
          <div className="col-span-full md:col-span-1">
            <Link href="#" className="flex font-bold items-center">
              <h3 className={ poppins.className + ' text-2xl'}>Kevin Gil</h3>
            </Link>
          </div>
          <div className="grid grid-cols-2 md:w-1/2">
            <div className="flex flex-col gap-2">
              <h3 className="font-bold text-lg">Socials</h3>
              <div>
                <Link href="https://github.com/kevingil" target="_blank" className="opacity-60 hover:opacity-100">
                  Github
                </Link>
              </div>

              <div>
                <Link href="https://linkedin.com/in/kevingil" target="_blank" className="opacity-60 hover:opacity-100">
                  LinkedIn
                </Link>
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <h3 className="font-bold text-lg">Navigate</h3>
              <div>
                <Link href="/blog" className="opacity-60 hover:opacity-100">
                  Blog
                </Link>
              </div>

              <div>
                <Link href="/contact" className="opacity-60 hover:opacity-100">
                  Contact
                </Link>
              </div>

              <div>
                <Link href="/about" className="opacity-60 hover:opacity-100">
                  About
                </Link>
              </div>
            </div>
          </div>
        </div>

        <Separator className="my-6" />
        <section className="">
          <h3 className="">
            <Link
              target="_blank"
              href="https://github.com/kevingil/blog"
              className="text-primary transition-all border-primary hover:border-b-2 ml-1"
            >
              kevingil/blog
            </Link>
          </h3>
        </section>
      </div>
    </footer>
  );
};
